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
	"sync"
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

	// Set to true for additional log information
	Debug bool

	// router is used to route incoming publish packets to subscribing
	// clients
	router *Router

	// statistics
	stat *ServerStats
}

// Run listens for tcp connections. Blocks until context is cancelled
// or accepting a connection fails. Accepting new connection can only
// be interrupted if listener has SetDeadline method.
func (s *Server) Run(ctx context.Context) error {
	if s.Debug {
		s.Log.SetFlags(s.Log.Flags() | log.Lshortfile)
	} else {
		s.Log.SetFlags(log.Flags()) // default
	}

	b := s.Binds[0]
	ln, err := net.Listen(b.URL.Scheme, b.URL.Host)
	if err != nil {
		return err
	}
	s.Log.Println("listen", ln.Addr())

	f := NewConnFeed()
	f.ServeConn = s.ServeConn
	f.Listener = ln
	f.AcceptTimeout = b.AcceptTimeout
	return f.Run(ctx)
}

func (s *Server) Stat() ServerStats {
	return *s.stat
}

// ServeConn handles the given remote connection. Blocks until
// receiver is done. Usually called in go routine.
func (s *Server) ServeConn(ctx context.Context, conn Connection) {
	// the server tracks active connections
	addr := conn.RemoteAddr()
	a := includePort(addr.String(), s.Debug)
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
		includePort(conn.RemoteAddr().String(), s.Debug),
	)
	logger.SetPrefix("ttsrv ")

	var m sync.Mutex
	var maxQoS uint8 = 0 // todo support QoS 1 and 2

	transmit = func(ctx context.Context, p mq.Packet) error {
		switch p := p.(type) {
		case *mq.ConnAck:
			p.SetMaxQoS(maxQoS)
			if v := p.AssignedClientID(); v != "" {
				logger.clientID = trimID(v, logger.maxLen)
			}
		}

		if s.Debug {
			logger.Print("out ", p, "\n", dumpPacket(p))
		} else {
			logger.Printf("out %v -> %s:%s", p, logger.remote, logger.clientID)
		}

		m.Lock()
		defer m.Unlock()
		if _, err := p.WriteTo(conn); err != nil {
			return err
		}

		switch p.(type) {
		case *mq.Disconnect:
			// close connection after Disconnect is send
			conn.Close()
		}
		return nil
	}

	in = func(ctx context.Context, p mq.Packet) error {

		if s.Debug {
			logger.Print("out ", p, "\n", dumpPacket(p))
		} else {
			logger.Printf("out %v -> %s:%s", p, logger.remote, logger.clientID)
		}

		if p, ok := p.(interface{ WellFormed() *mq.Malformed }); ok {
			if err := p.WellFormed(); err != nil {
				d := mq.NewDisconnect()
				d.SetReasonCode(mq.MalformedPacket)
				return transmit(ctx, d)
			}
		}

		switch p := p.(type) {
		case *mq.Connect:
			// todo should the ack be sent here?
			a := mq.NewConnAck()
			if id := p.ClientID(); id == "" {
				a.SetAssignedClientID(uuid.NewString())
			}
			return transmit(ctx, a)

		case *mq.Subscribe:
			a := mq.NewSubAck()
			a.SetPacketID(p.PacketID())
			for _, f := range p.Filters() {
				tf, err := tt.ParseTopicFilter(f.Filter())
				if err != nil {
					p := mq.NewDisconnect()
					p.SetReasonCode(mq.MalformedPacket)
					transmit(ctx, p)
					return nil
				}

				r := NewSubscription(tf, func(ctx context.Context, p *mq.Publish) error {
					return transmit(ctx, p)
				})
				s.router.AddRoute(r)
				// todo Subscribe.WellFormed fails if for any reason, though
				// here we want to set a reason code for each filter.
				// 3.9.3 SUBACK Payload
				a.AddReasonCode(mq.Success)
			}
			if err := transmit(ctx, a); err != nil {
				return err
			}

		case *mq.Publish:
			// Disconnect any attempts to publish exceeding qos.
			// Specified in section 3.3.1.2 QoS
			if p.QoS() > maxQoS {
				d := mq.NewDisconnect()
				d.SetReasonCode(mq.QoSNotSupported)
				return transmit(ctx, d)
			}
			return s.router.Handle(ctx, p)

		case *mq.Disconnect:
			return conn.Close()
		}
		return nil
	}
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
