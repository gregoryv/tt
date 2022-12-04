package tt

import (
	"context"
	"io"
	"log"
	"net"
	"os"
	"time"

	"github.com/gregoryv/mq"
)

// NewServer returns a server that binds to a random port.
func NewServer() *Server {
	return &Server{
		PoolSize: 100,
		Router:   NewRouter(),
		Logger:   log.New(os.Stdout, "ttsrv ", log.Flags()),
		stat:     NewServerStats(),
	}
}

type Server struct {
	// client has to send the initial connect packet
	ConnectTimeout time.Duration

	PoolSize uint16

	*Router

	*log.Logger

	// statistics
	stat *ServerStats
}

func (s *Server) Stat() ServerStats {
	return *s.stat
}

// AddConnection handles the given remote connection. Blocks until
// receiver is done. Usually called in go routine.
func (s *Server) AddConnection(ctx context.Context, conn Remote) {
	// the server tracks active connections
	s.Println("new conn", conn.RemoteAddr())
	s.stat.AddConn()
	defer func() {
		s.Logger.Println("del conn", conn)
		s.stat.RemoveConn()
	}()
	in, _ := s.CreateHandlers(conn)
	if err := NewReceiver(conn, in).Run(ctx); err != nil {
		s.Logger.Print(err)
	}
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
		Disconnect(conn),
		s.Router,
		clientIDmaker, quality, subscriber, pool, checker, logger,
	)
	return
}

type Remote interface {
	Conn
	RemoteAddr() net.Addr
}

func Disconnect(c io.Closer) Handler {
	return func(_ context.Context, p mq.Packet) error {
		switch p.(type) {
		case *mq.Disconnect:
			c.Close()
		}
		return nil
	}
}
