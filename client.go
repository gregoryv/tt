package tt

import (
	"context"
	"io"

	"github.com/gregoryv/mq"
)

func NewClient() *Client {
	return &Client{}
}

type Client struct {
	Conn io.ReadWriter // connection
}

func (c *Client) Connect(ctx context.Context) error {
	return nil
}

func (c *Client) Send(ctx context.Context, p mq.Packet) error {
	_, err := p.WriteTo(c.Conn)
	return err
}
