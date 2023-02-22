// Package tt provides components for writing mqtt-v5 clients.
package tt

import (
	"context"

	"github.com/gregoryv/mq"
)

type Middleware func(next Handler) Handler

// Handler handles a mqtt control packet
type Handler func(context.Context, mq.Packet) error
