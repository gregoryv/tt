package tt

import (
	"context"
	"errors"
	"io"
	"net"
	"os"
	"strings"
	"time"

	"github.com/gregoryv/mq"
)

// NewReceiver returns a receiver that reads packets from the reader
// and calls the handler made of number of middlewares and a final handler.
func NewReceiver(r io.Reader, h Handler) *Receiver {
	return &Receiver{
		wire:        r,
		handle:      h,
		readTimeout: 100 * time.Millisecond,
	}
}

type Receiver struct {
	wire        io.Reader
	handle      Handler
	readTimeout time.Duration
}

// Run begins reading incoming packets and forwards them to the
// configured handler.
func (r *Receiver) Run(ctx context.Context) error {
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
			if !strings.Contains(err.Error(), "use of closed network connection") {
				return err
			}
			return nil
		}
		// ignore error here, it's up to the user to configure a queue
		// where the first middleware handles any errors, eg. Logger
		_ = r.handle(ctx, p)
	}
}

func Start[T Runner](ctx context.Context, r T) (T, <-chan error) {
	c := make(chan error, 0)
	go func() {
		if err := r.Run(ctx); err != nil {
			if err != nil {
				c <- err
			}
			close(c)
		}
	}()
	return r, c
}

type Runner interface {
	Run(context.Context) error
}
