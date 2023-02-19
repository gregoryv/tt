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
// wip decouple listener from connection feeding
func NewConnFeed() *ConnFeed {
	return &ConnFeed{
		Up:        make(chan struct{}, 0),
		Logger:    log.New(os.Stderr, "tcp ", log.Flags()),
		ServeConn: func(context.Context, Connection) { /*noop*/ },
	}
}

type ConnFeed struct {
	// Up is closed when listener is running
	Up chan struct{}

	// Listener is set once run
	net.Listener

	// AddConnection handles new remote connections
	ServeConn func(context.Context, Connection)

	*log.Logger

	debug bool
}

func (f *ConnFeed) SetDebug(v bool) { f.debug = v }

// SetServer sets the server to which new connections should be added.
func (f *ConnFeed) SetServer(v interface {
	ServeConn(context.Context, Connection)
}) {
	f.ServeConn = v.ServeConn
}

// Run enables listener. Blocks until context is cancelled or
// accepting a connection fails. Accepting new connection can only be
// interrupted if listener has SetDeadline method.
func (f *ConnFeed) Run(ctx context.Context, l net.Listener, acceptTimeout time.Duration) error {
loop:
	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		// timeout Accept call so we don't block the loop
		if l, ok := l.(interface{ SetDeadline(time.Time) error }); ok {
			l.SetDeadline(time.Now().Add(acceptTimeout))
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
