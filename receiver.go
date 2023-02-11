package tt

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
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
			return err
		}
		// ignore most errors here, it's up to the user to configure a
		// queue where the first middleware handles any errors,
		// eg. Logger
		if err := r.handle(ctx, p); err != nil && errors.Is(err, StopReceiver) {
			return nil
		}
	}
}

// StopReceiver error can be returned by handlers to stop Receiver.Run
// from handling further packets.
var StopReceiver = fmt.Errorf("receiver stop")
