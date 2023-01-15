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
		router:   NewRouter(),
		log:      log.New(os.Stdout, "ttsrv ", log.Flags()),
		stat:     NewServerStats(),
	}
}

type Server struct {
	// client has to send the initial connect packet
	ConnectTimeout time.Duration

	PoolSize uint16

	router *Router

	log *log.Logger

	// statistics
	stat *ServerStats
}

func (s *Server) Stat() ServerStats {
	return *s.stat
}

func (s *Server) NewMemConn() Remote {
	conn := NewMemConn()
	go s.AddConnection(context.Background(), conn.Server())
	return conn.Client()
}

// AddConnection handles the given remote connection. Blocks until
// receiver is done. Usually called in go routine.
func (s *Server) AddConnection(ctx context.Context, conn Remote) {
	// the server tracks active connections
	a := conn.RemoteAddr()
	s.log.Printf("new conn %s://%s", a.Network(), a)
	s.stat.AddConn()
	defer func() {
		s.log.Printf("del conn %s://%s", a.Network(), a)
		s.stat.RemoveConn()
	}()
	in, _ := s.CreateHandlers(conn)
	if err := NewReceiver(conn, in).Run(ctx); err != nil {
		s.log.Printf("%T %v", err, err)
	}
}

// CreateHandlers returns in and out handlers for packets.
func (s *Server) CreateHandlers(conn Remote) (in, transmit Handler) {
	logger := NewLogger()
	logger.SetOutput(s.log.Writer())
	logger.SetRemote(conn.RemoteAddr().String())
	logger.SetPrefix("ttsrv ")

	pool := NewIDPool(s.PoolSize)
	disco := NewDisconnector(conn)
	subtransmit := CombineOut(Send(conn), logger, disco)
	quality := NewQualitySupport(subtransmit)
	transmit = CombineOut(Send(conn), logger, disco, quality, pool)

	clientIDmaker := NewClientIDMaker(subtransmit)
	checker := NewFormChecker(subtransmit)
	subscriber := NewSubscriber(s.router, transmit)

	in = CombineIn(
		s.router.Handle,
		disco, clientIDmaker, quality, subscriber, pool, checker, logger,
	)
	return
}

type Remote interface {
	io.ReadWriteCloser
	RemoteAddr() net.Addr
}

func NewDisconnector(conn io.Closer) *Disconnector {
	return &Disconnector{conn: conn}
}

type Disconnector struct {
	conn io.Closer
}

func (d *Disconnector) In(next Handler) Handler {
	return func(ctx context.Context, p mq.Packet) error {
		switch p.(type) {
		case *mq.Disconnect:
			d.conn.Close()
		}
		return next(ctx, p)
	}
}

func (d *Disconnector) Out(next Handler) Handler {
	return func(ctx context.Context, p mq.Packet) error {
		err := next(ctx, p) // send it first
		switch p.(type) {
		case *mq.Disconnect:
			d.conn.Close()
		}
		return err
	}
}
