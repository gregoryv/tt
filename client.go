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
)

// Client implements a mqtt-v5 client. Public fields should be set
// before calling Run.
type Client struct {
	// Server to connect to, defaults to tcp://localhost:1883
	Server *url.URL

	// Set to true to include more log output
	Debug bool

	// Outgoing packets use ids from 1..MaxPacketID, this limits the
	// number of packets in flight.
	MaxPacketID uint16

	// optional handler for incoming packets
	OnPacket func(context.Context, *Client, mq.Packet)

	// optional handler for client events
	OnEvent func(context.Context, *Client, Event)

	// optional ping interval for keeping connection alive
	KeepAlive time.Duration

	// for setting defaults
	once sync.Once

	// optional logger, leave empty for no logging
	*log.Logger

	transmit Handler // set by Run and used in Send
}

func (c *Client) setDefaults() {
	if c.Logger == nil {
		c.Logger = log.New(ioutil.Discard, "", log.Flags())
	}
	if c.Server == nil {
		u, _ := url.Parse("tcp://127.0.0.1:1883")
		c.Server = u
	}
}

func (c *Client) Run(ctx context.Context) error {
	c.once.Do(c.setDefaults)

	ctx, cancel := context.WithCancel(ctx)

	log := c.Logger
	debug := c.Debug
	log.Println("debug", debug)

	maxIDLen := uint(11)

	// dial server
	host := c.Server.Host
	log.Print("dial ", c.Server.String())
	conn, err := net.Dial(c.Server.Scheme, host)
	if err != nil {
		return err
	}

	// pool of packet ids for reuse
	pool := newIDPool(c.MaxPacketID)
	keepAlive := newKeepAlive(c.KeepAlive)
	log.Println("KeepAlive", c.KeepAlive)

	var m sync.Mutex
	c.transmit = func(ctx context.Context, p mq.Packet) error {
		// sync each outgoing packet
		m.Lock()
		defer m.Unlock()

		// set packet id if needed, only fails if pool is exhausted
		if err := pool.SetPacketID(ctx, p); err != nil {
			cancel()
			return err
		}

		// use client id
		switch p := p.(type) {
		case *mq.Connect:
			log.SetPrefix(trimID(p.ClientID(), maxIDLen) + " ")
			if v := p.KeepAlive(); v > 0 {
				keepAlive.SetInterval(v)
			}
		}

		// log just before sending
		log.Printf("out %s%s", p, dump(debug, p))

		// write packet on the wire
		_, err := p.WriteTo(conn)

		if err != nil {
			return err
		}
		// delay ping since we just send a packet
		keepAlive.packetSent <- struct{}{}

		return nil
	}
	keepAlive.transmit = c.transmit
	go keepAlive.run(ctx)

	recv := newReceiver(func(ctx context.Context, p mq.Packet) error {
		// set log prefix if client was assigned an id
		if p, ok := p.(*mq.ConnAck); ok {
			if v := p.AssignedClientID(); v != "" {
				log.SetPrefix(trimID(v, maxIDLen))
			}
		}
		// log incoming packet
		log.Printf("in %s%s", p, dump(debug, p))

		// check if malformed
		if p, ok := p.(interface{ WellFormed() *mq.Malformed }); ok {
			if err := p.WellFormed(); err != nil {
				d := mq.NewDisconnect()
				d.SetReasonCode(mq.MalformedPacket)
				c.transmit(ctx, d)
				return err
			}
		}

		// reuse packet id
		if p, ok := p.(mq.HasPacketID); ok {
			_ = pool.reuse(p.PacketID())
		}

		switch p := p.(type) {
		case *mq.ConnAck:
			// keep alive as the server instructs
			if v := p.ServerKeepAlive(); v > 0 {
				keepAlive.SetInterval(v)
			}

		case *mq.Publish:
			switch p.QoS() {
			case 0: // no ack is needed
			case 1:
				ack := mq.NewPubAck()
				ack.SetPacketID(p.PacketID())
				if err := c.transmit(ctx, ack); err != nil {
					return err
				}
			case 2:
				return fmt.Errorf("got QoS 2: unsupported ") // todo
			}
		}

		// finally let the application have it
		if c.OnPacket != nil {
			c.OnPacket(ctx, c, p)
		}
		return nil
	}, conn)

	if c.OnEvent != nil {
		c.OnEvent(ctx, c, EventClientUp)
	}
	return recv.Run(ctx)
}

// Send returns when the packet was successfully encoded on the wire.
// Returns ErrClientStopped if not running.
func (c *Client) Send(ctx context.Context, p mq.Packet) error {
	if c.transmit == nil {
		return ErrClientStopped
	}
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

// In checks if incoming packet has a packet ID, if so it's
// returned to the pool before next handler is called.
func (o *iDPool) In(next Handler) Handler {
	return func(ctx context.Context, p mq.Packet) error {
		if p, ok := p.(mq.HasPacketID); ok {
			_ = o.reuse(p.PacketID())
		}
		return next(ctx, p)
	}
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

// Out on outgoing packets, refs MQTT-2.2.1-3
func (o *iDPool) Out(next Handler) Handler {
	return func(ctx context.Context, p mq.Packet) error {
		if err := o.SetPacketID(ctx, p); err != nil {
			return err
		}
		return next(ctx, p)
	}
}

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
func newKeepAlive(interval time.Duration) *keepAlive {
	return &keepAlive{
		interval:   interval,
		tick:       time.NewTicker(time.Second),
		packetSent: make(chan struct{}, 1),
	}
}

// keepAlive implements client logic for keeping connection alive.
//
// See 3.1.2.10 Keep Alive
type keepAlive struct {
	transmit Handler
	interval time.Duration
	tick     *time.Ticker

	packetSent chan struct{}
}

func (k *keepAlive) SetInterval(v uint16) {
	k.interval = time.Duration(v) * time.Second
}

func (k *keepAlive) run(ctx context.Context) {
	last := time.Now()
	for {
		select {
		case <-ctx.Done():
			k.tick.Stop()
			return

		case <-k.packetSent:
			last = time.Now()

		case <-k.tick.C:
			if k.interval != 0 && time.Since(last) > k.interval {
				k.transmit(ctx, mq.NewPingReq())
			}
		}
	}
}
