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

func NewServer() *Server {
	return &Server{
		Bind:           ":", // random
		AcceptTimeout:  time.Millisecond,
		ConnectTimeout: 20 * time.Millisecond,
		PoolSize:       100,
		clients:        make(map[string]io.ReadWriter),
	}
}

type Server struct {
	Bind string

	AcceptTimeout time.Duration

	// client has to send the initial connect packet
	ConnectTimeout time.Duration

	// todo place this in a connections store
	clients map[string]io.ReadWriter

	PoolSize uint16
	pool     *IDPool // todo one / connection
}

// Run listens for tcp connections. Blocks until context is cancelled
// or accepting a connection fails. Accepting new connection can only
// be interrupted if listener has SetDeadline method.
func (s *Server) Run(l net.Listener, ctx Context) error {
	s.pool = NewIDPool(s.PoolSize)
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
			running := Start(ctx, s.CreateReceiver(ctx, conn))
			if err := <-running; err != nil {
				log.Print(err)
			}
		}()
	}
}

func (s *Server) CreateReceiver(ctx Context, conn io.ReadWriter) *Receiver {
	sender := NewSender(conn)
	logger := NewLogger(LevelInfo)
	out := s.pool.Out(logger.Out(sender.Out))

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
			fmt.Println("unhandled", p)
		}
		return nil
	}

	in := logger.In(s.pool.In(handler))
	return NewReceiver(in, conn)
}
