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
		poolSize: 100,
		router:   NewRouter(),
		log:      log.New(os.Stdout, "ttsrv ", log.Flags()),
		stat:     NewServerStats(),
	}
}

type Server struct {
	// client has to send the initial connect packet
	connectTimeout time.Duration
	// poolSize is the max packet id for each connection
	poolSize uint16
	// router is used to route incoming publish packets to subscribing
	// clients
	router *Router

	log *log.Logger

	// statistics
	stat *ServerStats
}

// SetConnectTimeout is the duration within which a client must send a
// mq.Connect packet or once a connection is opened.
func (s *Server) SetConnectTimeout(v time.Duration) { s.connectTimeout = v }

// SetPoolSize sets the total number of packets in transit for one
// connection. If e.g. set to 10, packets will be numbered 1 .. 10
func (s *Server) SetPoolSize(v uint16) {
	// +1 because 0 is not a valid packet ID
	s.poolSize = v + 1
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
	in, _ := s.createHandlers(conn)
	if err := NewReceiver(conn, in).Run(ctx); err != nil {
		s.log.Printf("%T %v", err, err)
	}
}

// createHandlers returns in and out handlers for packets.
func (s *Server) createHandlers(conn Remote) (in, transmit Handler) {
	logger := NewLogger()
	logger.SetOutput(s.log.Writer())
	logger.SetRemote(conn.RemoteAddr().String())
	logger.SetPrefix("ttsrv ")

	pool := NewIDPool(s.poolSize)
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
