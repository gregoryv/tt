package tt

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/url"
	"sync"
	"time"

	"github.com/gregoryv/mq"
	"github.com/gregoryv/tt/event"
)

func NewClient() *Client {
	return &Client{
		log:    log.New(ioutil.Discard, "", log.Flags()),
		server: "tcp://127.0.0.1:1883",
		app:    make(chan interface{}, 1),
	}
}

// Client implements a mqtt-v5 client.
type Client struct {
	// Server to connect to, defaults to tcp://localhost:1883
	server string

	// Set to true to include more log output
	debug bool

	// optional logger, leave empty for no logging
	log *log.Logger

	// Outgoing packets use ids from 1..MaxPacketID, this limits the
	// number of packets in flight.
	maxPacketID uint16

	// set by Run and used in Send
	transmit func(ctx context.Context, p mq.Packet) error

	app chan interface{}

	// synchronize Send
	m sync.Mutex
}

func (c *Client) SetServer(v string)      { c.server = v }
func (c *Client) SetDebug(v bool)         { c.debug = v }
func (c *Client) SetMaxPacketID(v uint16) { c.maxPacketID = v }
func (c *Client) SetLogger(v *log.Logger) { c.log = v }

func (c *Client) Run(ctx context.Context) error {
	err := c.run(ctx)
	c.app <- event.ClientStop{err}
	close(c.app)
	return err
}

// Events returns a channel used by client to inform application layer
// of packets and events. E.g. [event.ClientUp]
func (c *Client) Events() <-chan interface{} {
	return c.app
}

// run prepares and initiates transmit and receive logic of packets.
func (c *Client) run(ctx context.Context) error {

	s, err := url.Parse(c.server)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(ctx)
	maxIDLen := uint(11)

	// dial server
	c.log.Print("dial ", s.String())
	conn, err := net.Dial(s.Scheme, s.Host)
	if err != nil {
		return err
	}

	// pool of packet ids for reuse
	pool := newIDPool(c.maxPacketID)
	ping := newKeepAlive()

	transmit := func(ctx context.Context, p mq.Packet) error {
		// set packet id if needed, blocks if pool is exhausted
		if err := pool.SetPacketID(ctx, p); err != nil {
			cancel()
			return err
		}

		// use client id
		switch p := p.(type) {
		case *mq.Connect:
			c.log.SetPrefix(trimID(p.ClientID(), maxIDLen) + " ")
			if v := p.KeepAlive(); v > 0 {
				ping.SetInterval(v)
			}
		}

		// log just before sending
		c.log.Printf("out %s%s", p, dump(c.debug, p))

		// write packet on the wire
		if _, err := p.WriteTo(conn); err != nil {
			return err
		}

		// packet just sent, no need for a ping
		ping.delay()
		return nil
	}
	// set for use by method Client.Send
	c.transmit = transmit

	handle := func(ctx context.Context, p mq.Packet) {
		// set log prefix if client was assigned an id
		if p, ok := p.(*mq.ConnAck); ok {
			if v := p.AssignedClientID(); v != "" {
				c.log.SetPrefix(trimID(v, maxIDLen))
			}
		}
		// log incoming packet
		c.log.Printf("in %s%s", p, dump(c.debug, p))

		// check if malformed
		if p, ok := p.(interface{ WellFormed() *mq.Malformed }); ok {
			if err := p.WellFormed(); err != nil {
				d := mq.NewDisconnect()
				d.SetReasonCode(mq.MalformedPacket)
				_ = transmit(ctx, d)
			}
		}

		// reuse packet id
		if p, ok := p.(mq.HasPacketID); ok {
			_ = pool.reuse(p.PacketID())
		}

		switch p := p.(type) {
		case *mq.ConnAck:
			code := p.ReasonCode()
			switch {
			case code == mq.Success:
				c.app <- event.ClientConnect(0)
				// keep alive as the server instructs
				if v := p.ServerKeepAlive(); v > 0 {
					ping.SetInterval(v)
				}
				// client is connected start the ping routine
				go ping.run(ctx, transmit)

			case code >= 0x80:
				c.app <- event.ClientConnectFail(code.String())
			}

		case *mq.Publish:
			switch p.QoS() {
			case 0: // no ack is needed
			case 1:
				ack := mq.NewPubAck()
				ack.SetPacketID(p.PacketID())
				_ = transmit(ctx, ack)
			case 2:
				// todo supported QoS 2
				c.log.Print(fmt.Errorf("got QoS 2: unsupported "))
			}
		}

		// finally let the application have the packet
		c.app <- p
	}
	recv := newReceiver(handle, conn)

	c.app <- event.ClientUp(0)
	return recv.Run(ctx)
}

// Send returns when the packet was successfully encoded on the wire.
// Returns ErrClientStopped if not running. Send is safe to call
// concurrently.
func (c *Client) Send(ctx context.Context, p mq.Packet) error {
	if c.transmit == nil {
		return ErrClientStopped
	}
	// sync each outgoing packet
	c.m.Lock()
	defer c.m.Unlock()
	return c.transmit(ctx, p)
}

var ErrClientStopped = fmt.Errorf("Client stopped")

// newIDPool returns a iDPool of reusable id's from 1..max, 0 is not used
func newIDPool(max uint16) *iDPool {
	o := iDPool{
		nextTimeout: 3 * time.Second,
		max:         max,
		used:        make([]time.Time, max+1),
		values:      make(chan uint16, max),
	}
	for i := uint16(1); i <= max; i++ {
		o.values <- i
	}
	return &o
}

type iDPool struct {
	nextTimeout time.Duration
	max         uint16
	used        []time.Time // flags id that has been used in Out handler
	values      chan uint16
}

// reuse returns the given value to the pool, returns the reused value
// or 0 if ignored
func (o *iDPool) reuse(v uint16) uint16 {
	if v == 0 || v > o.max {
		return 0
	}
	if o.used[v].IsZero() {
		return 0
	}
	o.values <- v
	o.used[v] = zero
	return v
}

var zero = time.Time{}

func (o *iDPool) SetPacketID(ctx context.Context, p mq.Packet) error {
	if p, ok := p.(mq.HasPacketID); ok {
		switch p := p.(type) {
		case *mq.Publish:
			if p.QoS() > 0 && p.PacketID() == 0 {
				id, err := o.next(ctx)
				if err != nil {
					return err
				}
				p.SetPacketID(id)
			}

		case *mq.Subscribe:
			id, err := o.next(ctx)
			if err != nil {
				return err
			}
			p.SetPacketID(id)

		case *mq.Unsubscribe:
			id, err := o.next(ctx)
			if err != nil {
				return err
			}
			p.SetPacketID(id)
		}
	}
	return nil
}

// next returns the next available ID, blocks until one is available
// or context is canceled. next is safe for concurrent use by multiple
// goroutines.
func (o *iDPool) next(ctx context.Context) (uint16, error) {
	select {
	case <-ctx.Done():
		return 0, ctx.Err()

	case <-time.After(o.nextTimeout):
		return 0, ErrIDPoolEmpty

	case v := <-o.values:
		o.used[v] = time.Now()
		return v, nil
	}
}

var ErrIDPoolEmpty = fmt.Errorf("no available packet ids")

// sending pings within the given interval. The interval is rounded to
// seconds.
func newKeepAlive() *keepAlive {
	return &keepAlive{
		packetSent: make(chan struct{}, 1),
	}
}

// keepAlive implements client logic for keeping Connection alive.
//
// See 3.1.2.10 Keep Alive
type keepAlive struct {
	interval time.Duration

	packetSent chan struct{}
}

func (k *keepAlive) SetInterval(v uint16) {
	k.interval = time.Duration(v) * time.Second
}

func (k *keepAlive) delay() {
	k.packetSent <- struct{}{}
}

func (k *keepAlive) run(ctx context.Context, transmit errHandler) {
	last := time.Now()
	tick := time.NewTicker(time.Second)
	for {
		select {
		case <-ctx.Done():
			tick.Stop()
			return

		case <-k.packetSent:
			last = time.Now()

		case <-tick.C:
			if k.interval != 0 && time.Since(last) > k.interval {
				transmit(ctx, mq.NewPingReq())
			}
		}
	}
}
