package tt

import (
	"context"
	. "context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/url"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/gregoryv/mq"
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
	}
}

type Server struct {
	Bind string // todo support multiple binds, ie. tcp://:1883,ws://:8080
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
}

// Run listens for tcp connections. Blocks until context is cancelled
// or accepting a connection fails. Accepting new connection can only
// be interrupted if listener has SetDeadline method.
func (s *Server) Run(ctx Context) error {
	l := s.Listener
	if s.Listener == nil {
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

		// the server tracks active connections
		s.Println("accept", conn.RemoteAddr())
		go func() {
			in, _ := s.CreateHandlers(conn)
			_, done := Start(ctx, NewReceiver(conn, in))
			if err := <-done; err != nil {
				log.Print(err)
			}
		}()
	}
}

// CreateHandlers returns in and out handlers for packets.
func (s *Server) CreateHandlers(conn Remote) (in, out Handler) {
	logger := NewLogger()
	logger.SetOutput(s.Logger.Writer())
	logger.SetRemote(conn.RemoteAddr().String())

	pool := NewIDPool(s.PoolSize)
	out = pool.Out(logger.Out(Send(conn.(io.Writer))))

	handler := func(ctx context.Context, p mq.Packet) error {
		// todo these should probably just be implementations of Inner
		switch p := p.(type) {
		case *mq.Connect:
			// connect came in...
			a := mq.NewConnAck()
			id := p.ClientID()
			if id == "" {
				id = uuid.NewString()
			}
			// todo make sure it's uniq
			a.SetAssignedClientID(id)
			return out(ctx, a)

		case *mq.Publish:
			switch p.QoS() {
			case 1:
				a := mq.NewPubAck()
				a.SetPacketID(p.PacketID())
				return out(ctx, a)
			case 2:
				a := mq.NewPubRec()
				a.SetPacketID(p.PacketID())
				return out(ctx, a)
			}
			// todo route it
			s.Print("FAIL todo implement routing")
			return nil

		case *mq.PubRel:
			comp := mq.NewPubComp()
			comp.SetPacketID(p.PacketID())
			return out(ctx, comp)

		default:
			return fmt.Errorf("%v unhandled!", p)
		}
	}

	in = logger.In(CheckForm(pool.In(handler)))
	return
}

func (s *Server) URL() *url.URL {
	u, _ := url.Parse(
		fmt.Sprintf("%s://%s", s.Addr().Network(), s.Addr().String()),
	)
	return u
}

type Remote interface {
	io.Writer
	RemoteAddr() net.Addr
}
