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

// NewListener returns a listener for tcp connections on a random
// port. Each new connection is by handled in a go routine.
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

func (l *ConnFeed) SetDebug(v bool) { l.debug = v }

// SetServer sets the server to which new connections should be added.
func (s *ConnFeed) SetServer(v interface {
	AddConnection(context.Context, Connection)
}) {
	s.AddConnection = v.AddConnection
}

// Run enables listener. Blocks until context is cancelled or
// accepting a connection fails. Accepting new connection can only be
// interrupted if listener has SetDeadline method.
func (s *ConnFeed) Run(ctx context.Context) error {
	if s.Listener == nil {
		s.Println("listen", s.Bind)
		u, err := url.Parse(s.Bind)
		if err != nil {
			return err
		}
		ln, err := net.Listen(u.Scheme, u.Host)
		if err != nil {
			return err
		}
		s.Listener = ln
	}
	close(s.Up)
	return s.run(ctx, s.Listener)
}

func (s *ConnFeed) run(ctx context.Context, l net.Listener) error {
loop:
	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		// timeout Accept call so we don't block the loop
		if l, ok := l.(interface{ SetDeadline(time.Time) error }); ok {
			l.SetDeadline(time.Now().Add(s.AcceptTimeout))
		}
		conn, err := l.Accept()

		if errors.Is(err, os.ErrDeadlineExceeded) {
			continue loop
		}
		if err != nil {
			return err
		}

		go s.AddConnection(ctx, conn)
	}
}