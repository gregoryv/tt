package ttsrv

import (
	"context"

	"github.com/gregoryv/mq"
	"github.com/gregoryv/tt"
)

// NewIDPool returns a server side IDPool of reusable id's from 1..max
func NewIDPool(max uint16) *IDPool {
	return &IDPool{
		max: max,
	}
}

type IDPool struct {
	max uint16
}

// Out todo check if packets have ID set if required otherwise
// disconnect them.
func (o *IDPool) In(next tt.Handler) tt.Handler {
	return func(ctx context.Context, p mq.Packet) error {
		return next(ctx, p)
	}
}

// Out todo do outgoing packets don't need specific ID handling
func (o *IDPool) Out(next tt.Handler) tt.Handler {
	return func(ctx context.Context, p mq.Packet) error {
		return next(ctx, p)
	}
}
