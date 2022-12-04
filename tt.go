/*
Package tt provides components for writing mqtt-v5 clients and servers.
*/
package tt

import (
	"context"
	"io"

	"github.com/gregoryv/mq"
)

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

type PubHandler func(context.Context, *mq.Publish) error

func NoopHandler(_ context.Context, _ mq.Packet) error { return nil }
func NoopPub(_ context.Context, _ *mq.Publish) error   { return nil }

// Conn represents a connection
type Conn interface {
	io.ReadWriteCloser
}

// ----------------------------------------

// FormChecker rejects any malformed packet
type FormChecker struct{}

// CheckForm returns a handler that checks if a packet is well
// formed. The handler returns *mq.Malformed error without calling
// next if malformed.
func (f *FormChecker) In(next Handler) Handler {
	return func(ctx context.Context, p mq.Packet) error {
		if p, ok := p.(interface{ WellFormed() *mq.Malformed }); ok {
			if err := p.WellFormed(); err != nil {
				return err
			}
		}
		return next(ctx, p)
	}
}

func CombineIn(h Handler, v ...Inner) Handler {
	if len(v) == 0 {
		return h
	}
	n := len(v) - 1
	return v[n].In(CombineIn(h, v[:n]...))
}

func CombineOut(h Handler, v ...Outer) Handler {
	if len(v) == 0 {
		return h
	}
	n := len(v) - 1
	return v[n].Out(CombineOut(h, v[:n]...))
}
