package tt

import (
	"context"
	"io"
	"log"
	"net"
	"os"
	"time"
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

// AddConnection handles the given remote connection in a go routine.
func (s *Server) AddConnection(ctx context.Context, conn Remote) {

	// the server tracks active connections
	//s.Println("accept", conn.RemoteAddr())
	go func() {
		s.stat.AddConn()
		defer s.stat.RemoveConn()
		in, _ := s.CreateHandlers(conn)
		_, done := Start(ctx, NewReceiver(conn, in))
		if err := <-done; err != nil {
			s.Print(err)
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
