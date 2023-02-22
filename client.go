package tt

import (
	"context"
	"fmt"
	"net"
	"net/url"

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
}

func (c *Client) Run(ctx context.Context) error {
	// use middlewares and build your in/out queues with desired
	// features
	log := NewLogger()
	log.SetLogPrefix(c.ClientID)
	log.SetDebug(c.Debug)

	// dial server
	host := c.Server.String()
	log.Print("dial tcp://", host)
	conn, err := net.Dial("tcp", host)
	if err != nil {
		return err
	}

	_ = conn // wip use it
	return fmt.Errorf("Client.Run: todo")
}

func (c *Client) Send(ctx context.Context, p mq.Packet) error {
	return fmt.Errorf("Client.Send: todo")
}
