// Package tt provides components for writing mqtt-v5 clients.
package tt

import (
	"context"

	"github.com/gregoryv/mq"
)

type Middleware func(next Handler) Handler

// Handler handles a mqtt control packet
type Handler func(context.Context, mq.Packet) error

// Combine middlewares, from right to left, ending with handler.
func Combine(h Handler, v ...Middleware) Handler {
	if len(v) == 0 {
		return h
	}
	n := len(v) - 1
	return v[n](Combine(h, v[:n]...))
}
