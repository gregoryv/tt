package tt

import (
	"context"
	"errors"
	"io"
	"net"
	"os"
	"time"

	"github.com/gregoryv/mq"
)

// NewReceiver returns a receiver that reads packets from the reader
// and calls the handler. Handler can be nil.
func newReceiver(h Handler, r io.Reader) *receiver {
	return &receiver{
		wire:        r,
		handle:      h,
		readTimeout: 100 * time.Millisecond,
	}
}

type receiver struct {
	wire        io.Reader
	handle      Handler
	readTimeout time.Duration
}

// Run continuously handles next packet until context is cancelled
func (r *receiver) Run(ctx context.Context) error {
loop:
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if w, ok := r.wire.(net.Conn); ok {
			w.SetReadDeadline(time.Now().Add(r.readTimeout))
		}
		p, err := mq.ReadPacket(r.wire)
		if err != nil {
			if errors.Is(err, os.ErrDeadlineExceeded) {
				continue loop
			}
			return err
		}
		// ignore errors here, errors from handlers should never stop the receiver
		_ = r.handle(ctx, p)
	}
}
