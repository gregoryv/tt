/*
Package tt provides components for writing mqtt-v5 clients and servers.
*/
package tt

import (
	"context"
	"fmt"

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

var (
	// Client.IO is closed
	ErrNoConnection = fmt.Errorf("no connection")

	// Client has no receiver configured
	ErrUnsetReceiver = fmt.Errorf("unset receiver")

	// Client is not fully operational, ie. hasn't been started
	ErrNotRunning = fmt.Errorf("not running")

	// Settings cannot be changed
	ErrReadOnly = fmt.Errorf("read only")
)
