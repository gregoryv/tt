package tt

import (
	"context"
	. "context"
	"errors"
	"io"
	"log"
	"net"
	"net/url"
	"os"
	"time"
)

// NewServer returns a server that binds to a random port.
func NewServer() *Server {
	return &Server{
		Bind:           "tcp://:", // random
		AcceptTimeout:  time.Millisecond,
		ConnectTimeout: 20 * time.Millisecond,
		PoolSize:       100,
		Up:             make(chan struct{}, 0),
		Router:         NewRouter(),
		Logger:         log.New(os.Stdout, "ttsrv ", log.Flags()),
		stat:           NewServerStats(),
	}
}

type Server struct {
	// Scheme://[hostname]:port
	Bind string

	// Listener is set once run
	net.Listener

	// AcceptTimeout is used as deadline for new connections before
	// checking if context has been cancelled.
	AcceptTimeout time.Duration

	// client has to send the initial connect packet
	ConnectTimeout time.Duration

	PoolSize uint16

	// Up is closed when all listeners are running
	Up chan struct{}

	*Router

	*log.Logger

	// statistics
	stat *ServerStats
}

// Run listens for tcp connections. Blocks until context is cancelled
// or accepting a connection fails. Accepting new connection can only
// be interrupted if listener has SetDeadline method.
func (s *Server) Run(ctx Context) error {
	l := s.Listener
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
		l = ln
	}

	close(s.Up)
	return s.run(ctx, l)
}

func (s *Server) run(ctx Context, l net.Listener) error {
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
			// todo check what causes Accept to fail other than
			// timeout, guess not all errors should result in
			// server run to stop
			return err
		}

		s.AddConnection(ctx, conn)
	}
}

// AddConnection handles the given remote connection in a go routine.
func (s *Server) AddConnection(ctx context.Context, conn Remote) {
	// the server tracks active connections
	s.Println("accept", conn.RemoteAddr())
	go func() {
		s.stat.AddConn()
		defer s.stat.RemoveConn()
		in, _ := s.CreateHandlers(conn)
		_, done := Start(ctx, NewReceiver(conn, in))
		if err := <-done; err != nil {
			log.Print(err)
		}
	}()
}

// CreateHandlers returns in and out handlers for packets.
func (s *Server) CreateHandlers(conn Remote) (in, transmit Handler) {
	logger := NewLogger()
	logger.SetOutput(s.Logger.Writer())
	logger.SetRemote(conn.RemoteAddr().String())
	logger.SetPrefix("ttsrv ")

	pool := NewIDPool(s.PoolSize)
	subtransmit := CombineOut(Send(conn.(io.Writer)), logger)
	quality := NewQualitySupport(subtransmit)
	transmit = CombineOut(Send(conn.(io.Writer)), logger, quality, pool)

	clientIDmaker := NewClientIDMaker(subtransmit)
	checker := &FormChecker{}
	subscriber := NewSubscriber(s.Router, transmit)

	in = CombineIn(
		s.Router.Handle,
		clientIDmaker, quality, subscriber, pool, checker, logger,
	)
	return
}

type Remote interface {
	Conn
	RemoteAddr() net.Addr
}
