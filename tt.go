/*
Package tt provides components for writing mqtt-v5 clients and servers.
*/
package tt

import (
	"context"
	"io"

	"github.com/gregoryv/mq"
)

// Middleware control packet flow in both directions
type Middleware interface {
	Inner
	Outer
}

// Inner handles incoming packets
type Inner interface {
	In(next Handler) Handler
}

// Outer handles outgoing packets
type Outer interface {
	Out(next Handler) Handler
}

// Handler handles a mqtt control packet
type Handler func(context.Context, mq.Packet) error

func NoopHandler(_ context.Context, _ mq.Packet) error { return nil }
func NoopPub(_ context.Context, _ *mq.Publish) error   { return nil }

// Conn represents a connection
type Conn interface {
	io.ReadWriter
}

// ----------------------------------------

func CheckForm(next Handler) Handler {
	return func(ctx context.Context, p mq.Packet) error {
		if p, ok := p.(interface{ WellFormed() error }); ok {
			// todo mq.WellFormed must return a specific error
			if err := p.WellFormed(); err != nil {
				return err
			}
		}
		return next(ctx, p)
	}
}
