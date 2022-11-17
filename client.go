package tt

import (
	"context"
	"io"

	"github.com/gregoryv/mq"
	"github.com/gregoryv/nexus"
)

func NewClient() *Client {
	return &Client{}
}

type Client struct {
	// Dialer opens a network connection to some server
	Dialer

	netio io.ReadWriter // connection
}

func (c *Client) SetNetworkIO(v io.ReadWriter) { c.netio = v }

// Run activates this client. Should only be called once.
func (c *Client) Run(ctx context.Context) error {
	var err error
	next := nexus.NewStepper(&err)

	next.Stepf("dial: %w", func() {
		err = c.Dialer(ctx)
	})

	return err
}

// Send control packet to the server. In most cases this would be a
// *mq.Publish packet.
func (c *Client) Send(ctx context.Context, p mq.Packet) error {
	_, err := p.WriteTo(c.netio)
	return err
}
