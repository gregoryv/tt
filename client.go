package tt

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/gregoryv/mq"
)

type Client struct {
	// Public fields can all be modified before calling Run
	// Changing them afterwards should have no effect.

	// Server to connect to
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

	transmit Handler // set by Run and used in Send
}

func (c *Client) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)

	debug := c.Debug
	pingInterval := 30 * time.Second
	maxIDLen := uint(11)

	log := log.New(os.Stderr, "", log.Flags())

	// dial server
	host := c.Server.Host
	log.Print("dial ", c.Server.String())
	conn, err := net.Dial("tcp", host)
	if err != nil {
		return err
	}

	// pool of packet ids for reuse
	pool := newIDPool(c.MaxPacketID)
	keepAlive := newKeepAlive(pingInterval)
	go keepAlive.run(ctx)

	var m sync.Mutex
	c.transmit = func(ctx context.Context, p mq.Packet) error {
		// set packet id if needed
		if err := pool.SetPacketID(ctx, p); err != nil {
			cancel()
			return err
		}

		//
		switch p := p.(type) {
		case *mq.Connect:
			log.SetPrefix(trimID(p.ClientID(), maxIDLen))
			if v := p.KeepAlive(); v > 0 {
				keepAlive.interval = v
			}
		}

		// log just before sending
		log.Printf("in  %s%s", p, dump(debug, p))

		m.Lock()
		_, err := p.WriteTo(conn)
		m.Unlock()

		if err == nil {
			// delay ping since we just send a packet
			keepAlive.packetSent <- struct{}{}
		}
		return err
	}
	keepAlive.transmit = c.transmit

	recv := newReceiver(func(ctx context.Context, p mq.Packet) error {
		// log incoming packets
		switch p := p.(type) {
		case *mq.ConnAck:
			// update log prefix if client was assigned an id
			if v := p.AssignedClientID(); v != "" {
				log.SetPrefix(trimID(v, maxIDLen))
			}
		}
		// double spaces to align in/out. Usually this is not advised
		// but in here it really does aid when scanning for patterns
		// of packets.
		log.Printf("in  %s%s", p, dump(debug, p))

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

		// keep alive
		switch p := p.(type) {
		case *mq.ConnAck:
			// do as the server says
			if v := p.ServerKeepAlive(); v > 0 {
				keepAlive.interval = v
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
func (c *Client) Send(ctx context.Context, p mq.Packet) error {
	if c.transmit == nil {
		return ErrClientStopped
	}
	return c.transmit(ctx, p)
}

var ErrClientStopped = fmt.Errorf("Client stopped")

// gomerge src: idpool.go

// NewIDPool returns a iDPool of reusable id's from 1..max, 0 is not used
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

var zero = time.Time{}

// gomerge src: keepalive.go

// NewKeepAlive returns a middleware which keeps connection alive by
// sending pings within the given interval. The interval is rounded to
// seconds.
func newKeepAlive(interval time.Duration) *keepAlive {
	return &keepAlive{
		interval:   uint16(interval.Seconds()),
		tick:       time.NewTicker(time.Second),
		packetSent: make(chan struct{}, 1),
	}
}

// keepAlive implements client logic for keeping connection alive.
//
// See [3.1.2.10 Keep Alive]
//
// [3.1.2.10 Keep Alive]: https://docs.oasis-open.org/mqtt/mqtt/v5.0/os/mqtt-v5.0-os.html#_Keep_Alive_1
type keepAlive struct {
	transmit Handler
	interval uint16 // seconds
	tick     *time.Ticker

	packetSent   chan struct{}
	disconnected chan struct{}
}

func (k *keepAlive) SetTransmitter(v Handler) { k.transmit = v }

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
			// check if its time to send a ping
			if time.Since(last).Round(time.Second) == time.Duration(k.interval)*time.Second-time.Second {
				k.transmit(ctx, mq.NewPingReq())
			}
		}
	}
}
