package tt

import (
	"context"
	"errors"
	"io"
	"os"
	"time"

	"github.com/gregoryv/mq"
)

// newReceiver returns a receiver that reads packets from the reader
// and calls the handler. Handler can be nil.
func newReceiver(h Handler, r io.Reader) *receiver {
	return &receiver{
		wire:     r,
		handle:   h,
		deadline: 400 * time.Millisecond,
	}
}

type receiver struct {
	wire     io.Reader
	handle   Handler
	deadline time.Duration
}

// Run continuously handles next packet until context is cancelled
func (r *receiver) Run(ctx context.Context) error {
	type hasReadDeadline interface{ SetReadDeadline(time.Time) error }
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if w, ok := r.wire.(hasReadDeadline); ok {
			_ = w.SetReadDeadline(time.Now().Add(r.deadline))
		}
		p, err := mq.ReadPacket(r.wire)
		if err != nil {
			if errors.Is(err, os.ErrDeadlineExceeded) {
				continue
			}
			return err
		}
		r.handle(ctx, p)
	}
}
