package tt

import (
	"context"
	"errors"
	"fmt"
	"io"

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
	// Dialer opens a network connection to some server
	Dialer
	Handler
	*Logger

	out      Handler       // set by Run
	netio    io.ReadWriter // connection
	clientID string
}

func (c *Client) SetClientID(v string)         { c.clientID = v }
func (c *Client) SetNetworkIO(v io.ReadWriter) { c.netio = v }

// Run activates this client. Should only be called once.
func (c *Client) Run(ctx context.Context) error {
	var err error
	next := nexus.NewStepper(&err)

	next.Stepf("dial: %w", func() {
		err = c.Dialer(ctx)
	})

	// create receiver
	// todo these should be features
	var (
		pool = NewIDPool(100)
		out  = pool.Out(c.Logger.Out(Send(c.netio)))
		in   = c.Logger.In(pool.In(c.Handler))
	)
	c.out = out // to allow Client.Send

	_, running := Start(ctx, NewReceiver(in, c.netio))

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
	}
	return err
}

// Send control packet to the server. In most cases this would be a
// *mq.Publish packet.
func (c *Client) Send(ctx context.Context, p mq.Packet) error {
	return c.out(ctx, p)
}
