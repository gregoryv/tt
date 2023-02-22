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
	for {
		_, err := r.Next(ctx)

		switch {
		case err != nil:
			return err
		}
	}
}

// Next blocks until a packet is read and handled.
func (r *receiver) Next(ctx context.Context) (mq.Packet, error) {
loop:
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if w, ok := r.wire.(net.Conn); ok {
			w.SetReadDeadline(time.Now().Add(r.readTimeout))
		}
		p, err := mq.ReadPacket(r.wire)
		if err != nil {
			if errors.Is(err, os.ErrDeadlineExceeded) {
				continue loop
			}
			return nil, err
		}
		// ignore most errors here, it's up to the user to configure a
		// queue where the first middleware handles any errors,
		// eg. Logger
		if r.handle != nil {
			err = r.handle(ctx, p)
		}
		return p, err
	}
}
