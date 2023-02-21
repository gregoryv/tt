package ttsrv

import (
	"context"
	"errors"
	"log"
	"net"
	"os"
	"time"
)

// NewConnFeed returns a listener for tcp connections on a random
// port. Each new connection is by handled in a go routine.
func NewConnFeed() *ConnFeed {
	return &ConnFeed{
		AcceptTimeout: 200 * time.Millisecond,
		Logger:        log.New(os.Stderr, "tcp ", log.Flags()),
		ServeConn:     func(context.Context, Connection) { /*noop*/ },
	}
}

type ConnFeed struct {
	// Listener to watch
	net.Listener

	AcceptTimeout time.Duration

	// AddConnection handles new remote connections
	ServeConn func(context.Context, Connection)

	*log.Logger
}

// SetServer sets the server to which new connections should be added.
func (f *ConnFeed) SetServer(v interface {
	ServeConn(context.Context, Connection)
}) {
	f.ServeConn = v.ServeConn
}

// Run enables listener. Blocks until context is cancelled or
// accepting a connection fails. Accepting new connection can only be
// interrupted if listener has SetDeadline method.
func (f *ConnFeed) Run(ctx context.Context) error {
	l := f.Listener

loop:
	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		// set deadline allows to break the loop early should the
		// context be done
		if l, ok := l.(interface{ SetDeadline(time.Time) error }); ok {
			l.SetDeadline(time.Now().Add(f.AcceptTimeout))
		}
		conn, err := l.Accept()

		if errors.Is(err, os.ErrDeadlineExceeded) {
			continue loop
		}
		if err != nil {
			return err
		}

		go f.ServeConn(ctx, conn)
	}
}
