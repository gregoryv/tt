package ttsrv

import (
	"context"
	"errors"
	"log"
	"net"
	"net/url"
	"os"
	"time"
)

// NewConnFeed returns a listener for tcp connections on a random
// port. Each new connection is by handled in a go routine.
// wip decouple listener from connection feeding
func NewConnFeed() *ConnFeed {
	return &ConnFeed{
		Bind:          "tcp://:", // random
		Up:            make(chan struct{}, 0),
		AcceptTimeout: 100 * time.Millisecond,
		Logger:        log.New(os.Stderr, "tcp ", log.Flags()),
		AddConnection: func(context.Context, Connection) { /*noop*/ },
	}
}

type ConnFeed struct {
	// Scheme://[hostname]:port
	Bind string

	// Up is closed when listener is running
	Up chan struct{}

	// Listener is set once run
	net.Listener

	// AcceptTimeout is used as deadline for new connections before
	// checking if context has been cancelled.
	AcceptTimeout time.Duration

	// AddConnection handles new remote connections
	AddConnection func(context.Context, Connection)

	*log.Logger

	debug bool
}

func (f *ConnFeed) SetDebug(v bool) { f.debug = v }

// SetServer sets the server to which new connections should be added.
func (f *ConnFeed) SetServer(v interface {
	AddConnection(context.Context, Connection)
}) {
	f.AddConnection = v.AddConnection
}

// Run enables listener. Blocks until context is cancelled or
// accepting a connection fails. Accepting new connection can only be
// interrupted if listener has SetDeadline method.
func (f *ConnFeed) Run(ctx context.Context) error {
	if f.Listener == nil {
		f.Println("listen", f.Bind)
		u, err := url.Parse(f.Bind)
		if err != nil {
			return err
		}
		ln, err := net.Listen(u.Scheme, u.Host)
		if err != nil {
			return err
		}
		f.Listener = ln
	}
	close(f.Up)
	return f.run(ctx, f.Listener)
}

func (f *ConnFeed) run(ctx context.Context, l net.Listener) error {
loop:
	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		// timeout Accept call so we don't block the loop
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

		go f.AddConnection(ctx, conn)
	}
}
