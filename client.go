package tt

import (
	"context"

	"github.com/gregoryv/mq"
)

func NewClient() *Client {
	return &Client{}
}

type Client struct {
	Conn
}

func (c *Client) Send(ctx context.Context, p mq.Packet) error {
	_, err := p.WriteTo(c.Conn)
	return err
}
