package tt

import (
	"context"

	"github.com/google/uuid"
	"github.com/gregoryv/mq"
)

func NewClientIDMaker(transmit Handler) *ClientIDMaker {
	return &ClientIDMaker{
		transmit: transmit,
	}
}

type ClientIDMaker struct {
	transmit Handler
}

func (c *ClientIDMaker) In(next Handler) Handler {
	return func(ctx context.Context, p mq.Packet) error {
		switch p := p.(type) {
		case *mq.Connect:
			a := mq.NewConnAck()
			if id := p.ClientID(); id == "" {
				a.SetAssignedClientID(uuid.NewString())
			}
			return c.transmit(ctx, a)
		}
		return next(ctx, p)
	}
}
