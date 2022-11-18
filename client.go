package tt

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"

	"github.com/gregoryv/mq"
	"github.com/gregoryv/nexus"
)

func NewClient() *Client {
	return &Client{
		Handler: NoopHandler,
		Logger:  NewLogger(LevelInfo),
	}
}

type Client struct {
	Handler
	*Logger

	out   Handler       // set by Run
	netio io.ReadWriter // connection

	clientID   string
	serverAddr *url.URL
}

func (c *Client) SetServerAddr(v *url.URL) { c.serverAddr = v }
func (c *Client) SetClientID(v string)     { c.clientID = v }

// Run activates this client. Should only be called once.
func (c *Client) Run(ctx context.Context) error {
	var err error
	next := nexus.NewStepper(&err)

	next.Stepf("dial: %w", func() {
		a := c.serverAddr
		c.netio, err = net.Dial(a.Scheme, a.Host)
	})

	// create receiver
	// todo these should be features
	var (
		pool = NewIDPool(100)
		out  = pool.Out(c.Logger.Out(Send(c.netio)))
	)
	c.out = out // to allow Client.Send

	_, running := Start(ctx, NewReceiver(c.netio, c.Logger, pool, c.Handler))

	next.Stepf("connect: %w", func() {
		p := mq.NewConnect()
		p.SetClientID(c.clientID)
		err = out(ctx, p)
	})

	select {
	case <-ctx.Done():
		return nil
	case err := <-running:
		if errors.Is(err, io.EOF) {
			// todo if we want a reconnect feature, this needs to be
			// handled
			return fmt.Errorf("remote disconnect")
		}
		return err
	}
	return err
}

// Send control packet to the server. In most cases this would be a
// *mq.Publish packet.
func (c *Client) Send(ctx context.Context, p mq.Packet) error {
	return c.out(ctx, p)
}
