package tt

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"sync"

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

	transmit Handler // set by Run and used in Send
}

func (c *Client) Run(ctx context.Context) error {
	// use middlewares and build your in/out queues with desired
	// features
	debug := c.Debug

	log := NewLogger()
	log.SetLogPrefix(c.ClientID)
	log.SetDebug(debug)

	// dial server
	host := c.Server.String()
	log.Print("dial tcp://", host)
	conn, err := net.Dial("tcp", host)
	if err != nil {
		return err
	}

	// pool of packet ids for reuse
	pool := NewIDPool(c.MaxPacketID)

	var m sync.Mutex
	c.transmit = func(ctx context.Context, p mq.Packet) error {
		// set packet id if needed
		if err := pool.SetPacketID(ctx, p); err != nil {
			return err
		}

		m.Lock()
		_, err := p.WriteTo(conn)
		m.Unlock()
		return err
	}

	receiver := NewReceiver(func(ctx context.Context, p mq.Packet) error {
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

		// return packet id to pool
		if p, ok := p.(mq.HasPacketID); ok {
			_ = pool.reuse(p.PacketID())
		}

		// wip implement receiving end of client
		return fmt.Errorf("receiver handler: todo")
	}, conn)

	return receiver.Run(ctx)
}

func (c *Client) Send(ctx context.Context, p mq.Packet) error {
	if c.transmit == nil {
		return ErrClientStopped
	}
	return c.transmit(ctx, p)
}

var ErrClientStopped = fmt.Errorf("Client stopped")
