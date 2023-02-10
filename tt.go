// Package tt provides components for writing mqtt-v5 clients.
package tt

import (
	"context"
	"io"
	"net"

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

// ----------------------------------------

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

type Remote interface {
	io.ReadWriteCloser
	RemoteAddr() net.Addr
}
