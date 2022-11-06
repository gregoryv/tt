package tt

import (
	"context"
	. "context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/gregoryv/mq"
)

// NewServer returns a server that binds to a random port.
func NewServer() *Server {
	return &Server{
		TCP: &TCP{
			Bind: ":", // random
		},
		AcceptTimeout:  time.Millisecond,
		ConnectTimeout: 20 * time.Millisecond,
		PoolSize:       100,
		Up:             make(chan struct{}, 0),
	}
}

type Server struct {
	*TCP

	// AcceptTimeout is used as deadline for new connections before
	// checking if context has been cancelled.
	AcceptTimeout time.Duration

	// client has to send the initial connect packet
	ConnectTimeout time.Duration

	PoolSize uint16

	// Up is closed when all listeners are running
	Up chan struct{}
}

type TCP struct {
	// [hostname]:[port] where server listens for connections, use ':'
	// for random port.
	Bind string

	Listener
}

type Listener interface {
	net.Listener
}

// Run listens for tcp connections. Blocks until context is cancelled
// or accepting a connection fails. Accepting new connection can only
// be interrupted if listener has SetDeadline method.
func (s *Server) Run(ctx Context) error {
	l := s.TCP.Listener
	if l == nil {
		var err error
		l, err = net.Listen("tcp", s.Bind)
		if err != nil {
			return err
		}
		s.Listener = l
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
		go func() {
			in, _ := s.CreateHandlers(conn)
			_, done := Start(ctx, NewReceiver(in, conn))
			if err := <-done; err != nil {
				log.Print(err)
			}
		}()
	}
}

func (s *Server) CreateHandlers(conn io.ReadWriter) (in, out Handler) {
	logger := NewLogger(LevelInfo)
	pool := NewIDPool(s.PoolSize)
	out = pool.Out(logger.Out(Send(conn)))

	handler := func(ctx context.Context, p mq.Packet) error {
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
			return nil

		case *mq.PubRel:
			comp := mq.NewPubComp()
			comp.SetPacketID(p.PacketID())
			return out(ctx, comp)

		default:
			return fmt.Errorf("unhandled %v", p)
		}
	}

	in = logger.In(CheckForm(pool.In(handler)))
	return
}
