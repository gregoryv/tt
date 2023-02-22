package tt

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"sync"
	"time"

	"github.com/gregoryv/mq"
)

func NewClient() *Client {
	return &Client{}
}

type Client struct {
	// Public fields can all be modified before calling Run
	// Changing them afterwards should have no effect.

	// Server to connect to
	Server *url.URL

	ClientID string

	Debug bool

	MaxPacketID uint16

	// wip better let application decide when sending Connect
	KeepAlive uint16 // seconds

	transmit Handler // set by Run and used in Send
}

func (c *Client) Run(ctx context.Context, app Handler) error {
	// use middlewares and build your in/out queues with desired
	// features
	debug := c.Debug
	pingInterval := time.Duration(c.KeepAlive) * time.Second

	log := NewLogger()
	log.SetLogPrefix(c.ClientID)
	log.SetDebug(debug)

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
			return err
		}

		//
		switch p := p.(type) {
		case *mq.Connect:
			p.SetKeepAlive(uint16(pingInterval / time.Second))
		}

		// log just before sending
		if debug {
			log.Print("in  ", p, "\n", dumpPacket(p))
		} else {
			log.Print("in  ", p)
		}

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
				log.SetLogPrefix(v)
			}
		}
		// double spaces to align in/out. Usually this is not advised
		// but in here it really does aid when scanning for patterns
		// of packets.
		if debug {
			log.Print("in  ", p, "\n", dumpPacket(p))
		} else {
			log.Print("in  ", p)
		}

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
		return app(ctx, p)
	}, conn)

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
