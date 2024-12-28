package tt

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/gregoryv/mq"
)

// newReceiver returns a receiver that reads packets from the reader
// and calls the handler.
func newReceiver(h handlerFunc) *receiver {
	return &receiver{
		handle:   h,
		deadline: 400 * time.Millisecond,
	}
}

// handlerFunc handles a mqtt control packet
type handlerFunc func(context.Context, mq.Packet)

type receiver struct {
	handle   handlerFunc
	deadline time.Duration
}

// Run handles packets until context is cancelled.
func (c *receiver) Run(ctx context.Context, conn Connection) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		conn.SetReadDeadline(time.Now().Add(c.deadline))

		p, err := mq.ReadPacket(conn)
		if err != nil {
			if errors.Is(err, os.ErrDeadlineExceeded) {
				continue
			}
			return err
		}
		c.handle(ctx, p)
	}
}
