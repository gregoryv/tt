// Package ttsrv provides a mqtt-v5 server
package ttsrv

import (
	"context"
	"io"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/gregoryv/mq"
	"github.com/gregoryv/tt"
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

// AddConnection handles the given remote connection. Blocks until
// receiver is done. Usually called in go routine.
func (s *Server) AddConnection(ctx context.Context, conn tt.Remote) {
	// the server tracks active connections
	a := conn.RemoteAddr()
	s.log.Printf("new conn %s://%s", a.Network(), a)
	s.stat.AddConn()
	defer func() {
		s.log.Printf("del conn %s://%s", a.Network(), a)
		s.stat.RemoveConn()
	}()
	in, _ := s.createHandlers(conn)
	if err := tt.NewReceiver(conn, in).Run(ctx); err != nil {
		s.log.Printf("%T %v", err, err)
	}
}

// createHandlers returns in and out handlers for packets.
func (s *Server) createHandlers(conn tt.Remote) (in, transmit tt.Handler) {
	logger := tt.NewLogger()
	logger.SetOutput(s.log.Writer())
	logger.SetRemote(conn.RemoteAddr().String())
	logger.SetPrefix("ttsrv ")

	pool := tt.NewIDPool(s.poolSize) // todo replace this with server side pool
	disco := NewDisconnector(conn)
	subtransmit := tt.CombineOut(tt.Send(conn), logger, disco)
	quality := tt.NewQualitySupport(subtransmit)
	transmit = tt.CombineOut(tt.Send(conn), logger, disco, quality, pool)

	clientIDmaker := NewClientIDMaker(subtransmit)
	checker := NewFormChecker(subtransmit)
	subscriber := NewSubscriber(s.router, transmit)

	in = tt.CombineIn(
		s.router.Handle,
		disco, clientIDmaker, quality, subscriber, pool, checker, logger,
	)
	return
}

func NewDisconnector(conn io.Closer) *Disconnector {
	return &Disconnector{conn: conn}
}

type Disconnector struct {
	conn io.Closer
}

func (d *Disconnector) In(next tt.Handler) tt.Handler {
	return func(ctx context.Context, p mq.Packet) error {
		switch p.(type) {
		case *mq.Disconnect:
			d.conn.Close()
		}
		return next(ctx, p)
	}
}

func (d *Disconnector) Out(next tt.Handler) tt.Handler {
	return func(ctx context.Context, p mq.Packet) error {
		err := next(ctx, p) // send it first
		switch p.(type) {
		case *mq.Disconnect:
			d.conn.Close()
		}
		return err
	}
}

func NewClientIDMaker(transmit tt.Handler) *ClientIDMaker {
	return &ClientIDMaker{
		transmit: transmit,
	}
}

type ClientIDMaker struct {
	transmit tt.Handler
}

func (c *ClientIDMaker) In(next tt.Handler) tt.Handler {
	return func(ctx context.Context, p mq.Packet) error {
		switch p := p.(type) {
		case *mq.Connect:
			a := mq.NewConnAck()
			if id := p.ClientID(); id == "" {
				a.SetAssignedClientID(uuid.NewString())
			}
			return c.transmit(ctx, a)
		}
		return next(ctx, p)
	}
}

func NewFormChecker(transmit tt.Handler) *FormChecker {
	return &FormChecker{
		transmit: transmit,
	}
}

// FormChecker rejects any malformed packet
type FormChecker struct {
	transmit tt.Handler
}

// CheckForm returns a handler that checks if a packet is well
// formed. The handler returns *mq.Malformed error without calling
// next if malformed.
func (f *FormChecker) In(next tt.Handler) tt.Handler {
	return func(ctx context.Context, p mq.Packet) error {
		if p, ok := p.(interface{ WellFormed() *mq.Malformed }); ok {
			if err := p.WellFormed(); err != nil {
				d := mq.NewDisconnect()
				d.SetReasonCode(mq.MalformedPacket)
				f.transmit(ctx, d)
				return err
			}
		}
		return next(ctx, p)
	}
}

type PubHandler func(context.Context, *mq.Publish) error
