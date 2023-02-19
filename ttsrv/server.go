// Package ttsrv provides a mqtt-v5 server
package ttsrv

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gregoryv/mq"
	"github.com/gregoryv/tt"
)

// NewServer returns a server that binds to a random port.
func NewServer() *Server {
	tcpRandom, _ := NewBindConf("tcp://localhost:", "500ms")
	s := &Server{
		Binds:          []*BindConf{tcpRandom},
		ConnectTimeout: 200 * time.Millisecond,
		PoolSize:       100,

		router: NewRouter(),
		Log:    log.New(os.Stderr, "ttsrv ", log.Flags()),
		stat:   NewServerStats(),
	}
	return s
}

type Server struct {
	Binds []*BindConf

	// client has to send the initial connect packet
	ConnectTimeout time.Duration

	// Max packet id for each connection, ids range from 1..PoolSize
	PoolSize uint16

	Log *log.Logger

	// router is used to route incoming publish packets to subscribing
	// clients
	router *Router

	// statistics
	stat *ServerStats

	debug bool
}

// Run listens for tcp connections. Blocks until context is cancelled
// or accepting a connection fails. Accepting new connection can only
// be interrupted if listener has SetDeadline method.
func (s *Server) Run(ctx context.Context) error {
	b := s.Binds[0]
	ln, err := net.Listen(b.URL.Scheme, b.URL.Host)
	if err != nil {
		return err
	}
	s.Log.Println("listen", ln.Addr())

	f := NewConnFeed()
	f.ServeConn = s.ServeConn
	f.SetDebug(s.debug)
	return f.Run(ctx, ln, b.AcceptTimeout)
}

func (s *Server) SetDebug(v bool) {
	s.debug = v
	if v {
		s.Log.SetFlags(s.Log.Flags() | log.Lshortfile)
	} else {
		s.Log.SetFlags(log.Flags()) // default
	}
}

func (s *Server) Stat() ServerStats {
	return *s.stat
}

// ServeConn handles the given remote connection. Blocks until
// receiver is done. Usually called in go routine.
func (s *Server) ServeConn(ctx context.Context, conn Connection) {
	// the server tracks active connections
	addr := conn.RemoteAddr()
	a := includePort(addr.String(), s.debug)
	connstr := fmt.Sprintf("conn %s://%s", addr.Network(), a)
	s.Log.Println("new", connstr)
	s.stat.AddConn()
	defer func() {
		s.Log.Println("del", connstr)
		s.stat.RemoveConn()
	}()
	in, _ := s.createHandlers(conn)
	// ignore error here, the connection is done
	_ = tt.NewReceiver(in, conn).Run(ctx)
}

// createHandlers returns in and out handlers for packets.
func (s *Server) createHandlers(conn Connection) (in, transmit tt.Handler) {
	// Note! ttsrv.Logger
	logger := NewLogger()
	logger.SetOutput(s.Log.Writer())
	logger.SetRemote(
		includePort(conn.RemoteAddr().String(), s.debug),
	)
	logger.SetPrefix("ttsrv ")

	// This design is hard to reason about as each middleware
	// may or may not stop the processing and you cannot see that
	// from here.

	// Try using a simpler form with one handler and perhaps a nexus

	pool := NewIDPool(s.PoolSize)
	disco := NewDisconnector(conn)
	sender := tt.Send(conn)
	// subtransmit is used for features sending acks
	subtransmit := tt.Combine(sender, logger.Out, disco.Out)

	quality := NewQualitySupport(subtransmit)
	transmit = tt.Combine(
		sender,
		// note! there is no check for malformed packets here for now
		// so the server could send them.

		// log just before sending the packet
		logger.Out,

		// close connection after Disconnect is send
		disco.Out,

		// todo do we have to check outgoing if incoming are already checked?
		quality.Out,

		// handle number of packets in flight
		pool.Out,
	)

	in = tt.Combine(
		s.router.Handle,
		disco.In,
		NewClientIDMaker(subtransmit).In,

		// make sure only supported QoS packets
		quality.In,

		// handle subscriptions in the router
		NewSubscriber(s.router, transmit).In,

		// handle packet ids correctly
		pool.In,

		// handle malformed packets early
		NewFormChecker(subtransmit).In,

		// log incoming packets first as they might change
		logger.In,
	)
	return
}

func includePort(addr string, yes bool) string {
	if yes {
		return addr
	}
	if i := strings.Index(addr, ":"); i > 0 {
		return addr[:i]
	}
	return addr
}

func NewDisconnector(conn io.Closer) *Disconnector {
	return &Disconnector{conn: conn}
}

// Disconnector handles Disconnect packets
type Disconnector struct {
	conn io.Closer
}

// In closes the connection and then calls the next handler
func (d *Disconnector) In(next tt.Handler) tt.Handler {
	return func(ctx context.Context, p mq.Packet) error {
		switch p.(type) {
		case *mq.Disconnect:
			d.conn.Close()
		}
		return next(ctx, p)
	}
}

// Out calls the next handler first, so the packet is actually send
// followed by a closing of the connection.
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

// In sends acs for incoming Connect packets.
func (c *ClientIDMaker) In(next tt.Handler) tt.Handler {
	return func(ctx context.Context, p mq.Packet) error {
		switch p := p.(type) {
		case *mq.Connect:
			// todo should the ack be sent here?
			a := mq.NewConnAck()
			if id := p.ClientID(); id == "" {
				a.SetAssignedClientID(uuid.NewString())
			}
			_ = c.transmit(ctx, a)
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

type Connection interface {
	io.ReadWriteCloser
	RemoteAddr() net.Addr
}
