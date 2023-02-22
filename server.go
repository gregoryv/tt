package tt

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/gregoryv/mq"
)

// NewServer returns a server that binds to a random port.
func NewServer() *Server {
	tcpRandom, _ := newBindConf("tcp://localhost:", "500ms")
	s := &Server{
		Binds:          []*bindConf{tcpRandom},
		ConnectTimeout: 200 * time.Millisecond,
		PoolSize:       100,

		router: newRouter(),
		Log:    log.New(os.Stderr, "ttsrv ", log.Flags()),
		stat:   newServerStats(),
	}
	return s
}

type Server struct {
	Binds []*bindConf

	// client has to send the initial connect packet
	ConnectTimeout time.Duration

	// Max packet id for each connection, ids range from 1..PoolSize
	PoolSize uint16

	Log *log.Logger

	// Set to true for additional log information
	Debug bool

	// wip
	OnEvent func(context.Context, *Server, Event)

	// router is used to route incoming publish packets to subscribing
	// clients
	router *router

	// statistics
	stat *serverStats
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

	f := newConnFeed()
	f.ServeConn = s.ServeConn
	f.Listener = ln
	f.AcceptTimeout = b.AcceptTimeout
	if s.OnEvent != nil {
		s.OnEvent(ctx, s, EventServerUp)
	}
	return f.Run(ctx)
}

const (
	EventServerUp Event = iota + lastClientEvent
)

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

	var m sync.Mutex
	var maxQoS uint8 = 0 // todo support QoS 1 and 2
	var maxIDLen uint = 11
	var (
		clientID string
		shortID  string
		remote   = includePort(conn.RemoteAddr().String(), s.Debug)
	)

	transmit := func(ctx context.Context, p mq.Packet) error {
		switch p := p.(type) {
		case *mq.ConnAck:
			p.SetMaxQoS(maxQoS)
		}

		if s.Debug {
			s.Log.Print("out ", p, "\n", dumpPacket(p))
		} else {
			s.Log.Printf("out %v -> %s:%s", p, remote, shortID)
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

	in := func(ctx context.Context, p mq.Packet) error {
		switch p := p.(type) {
		case *mq.Connect:
			// generate a client id before any logging
			clientID = p.ClientID()
			if clientID == "" {
				clientID = uuid.NewString()
			}
			shortID = trimID(clientID, maxIDLen)
		}

		if s.Debug {
			s.Log.Print("in  ", p, "\n", dumpPacket(p))
		} else {
			s.Log.Printf("in %v <- %s:%s", p, remote, shortID)
		}

		if p, ok := p.(interface{ WellFormed() *mq.Malformed }); ok {
			if err := p.WellFormed(); err != nil {
				d := mq.NewDisconnect()
				d.SetReasonCode(mq.MalformedPacket)
				return transmit(ctx, d)
			}
		}

		switch p := p.(type) {
		case *mq.PingReq:

			// 3.12.4-1 The Server MUST send a PINGRESP packet in
			// response to a PINGREQ packet
			return transmit(ctx, mq.NewPingResp())

		case *mq.Connect:
			a := mq.NewConnAck()
			if p.ClientID() == "" {
				a.SetAssignedClientID(clientID)
			}

			return transmit(ctx, a)

		case *mq.Subscribe:
			a := mq.NewSubAck()
			a.SetPacketID(p.PacketID())
			for _, f := range p.Filters() {
				tf, err := parseTopicFilter(f.Filter())
				if err != nil {
					p := mq.NewDisconnect()
					p.SetReasonCode(mq.MalformedPacket)
					transmit(ctx, p)
					return nil
				}

				r := newSubscription(tf, func(ctx context.Context, p *mq.Publish) error {
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

	// ignore error here, the connection is done
	_ = newReceiver(in, conn).Run(ctx)
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

type pubHandler func(context.Context, *mq.Publish) error

type Connection interface {
	io.ReadWriteCloser
	RemoteAddr() net.Addr
}

// gomerge src: bindconf.go

func newBindConf(uri, acceptTimeout string) (*bindConf, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}
	d, err := time.ParseDuration(acceptTimeout)
	if err != nil {
		return nil, err
	}

	return &bindConf{
		URL:           u,
		AcceptTimeout: d,
	}, nil
}

type bindConf struct {
	*url.URL
	AcceptTimeout time.Duration
	Debug         bool
}

// gomerge src: connfeed.go

// NewConnFeed returns a listener for tcp connections on a random
// port. Each new connection is by handled in a go routine.
func newConnFeed() *connFeed {
	return &connFeed{
		AcceptTimeout: 200 * time.Millisecond,
		Logger:        log.New(os.Stderr, "tcp ", log.Flags()),
		ServeConn:     func(context.Context, Connection) { /*noop*/ },
	}
}

type connFeed struct {
	// Listener to watch
	net.Listener

	AcceptTimeout time.Duration

	// AddConnection handles new remote connections
	ServeConn func(context.Context, Connection)

	*log.Logger
}

// SetServer sets the server to which new connections should be added.
func (f *connFeed) SetServer(v interface {
	ServeConn(context.Context, Connection)
}) {
	f.ServeConn = v.ServeConn
}

// Run enables listener. Blocks until context is cancelled or
// accepting a connection fails. Accepting new connection can only be
// interrupted if listener has SetDeadline method.
func (f *connFeed) Run(ctx context.Context) error {
	l := f.Listener

loop:
	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		// set deadline allows to break the loop early should the
		// context be done
		if l, ok := l.(interface{ SetDeadline(time.Time) error }); ok {
			l.SetDeadline(time.Now().Add(f.AcceptTimeout))
		}
		conn, err := l.Accept()

		if errors.Is(err, os.ErrDeadlineExceeded) {
			continue loop
		}
		if err != nil {
			return err
		}

		go f.ServeConn(ctx, conn)
	}
}

// gomerge src: router.go

// NewRouter returns a router for handling the given subscriptions.
func newRouter(v ...*subscription) *router {
	return &router{
		subs: v,
		log:  log.New(log.Writer(), "router ", log.Flags()),
	}
}

type router struct {
	subs []*subscription

	log *log.Logger
}

func (r *router) String() string {
	return plural(len(r.subs), "subscription")
}

func (r *router) AddRoute(v *subscription) {
	r.subs = append(r.subs, v)
}

// In forwards routes mq.Publish packets by topic name.
func (r *router) Handle(ctx context.Context, p mq.Packet) error {
	switch p := p.(type) {
	case *mq.Publish:
		// naive implementation looping over each route, improve at
		// some point
		for _, s := range r.subs {
			if _, ok := s.Match(p.TopicName()); ok {
				for _, h := range s.handlers {
					// maybe we'll have to have a different routing mechanism for
					// client side handling subscriptions compared to server side.
					// As server may have to adapt packages before sending and
					// there will be a QoS on each subscription that we need to consider.
					if err := h(ctx, p); err != nil {
						r.log.Println("handle", p, err)
					}
				}
			}
		}
	}
	return ctx.Err()
}

func plural(v int, word string) string {
	if v > 1 {
		word = word + "s"
	}
	return fmt.Sprintf("%v %s", v, word)
}

// gomerge src: srvstats.go

func newServerStats() *serverStats {
	return &serverStats{}
}

type serverStats struct {
	ConnCount  int64
	ConnActive int64
}

func (s *serverStats) AddConn() {
	atomic.AddInt64(&s.ConnCount, 1)
	atomic.AddInt64(&s.ConnActive, 1)
}

func (s *serverStats) RemoveConn() {
	atomic.AddInt64(&s.ConnActive, -1)
}

// MustNewSubscription panics on bad filter
func mustNewSubscription(filter string, handlers ...pubHandler) *subscription {
	tf, err := parseTopicFilter(filter)
	if err != nil {
		panic(err.Error())
	}
	return newSubscription(tf, handlers...)
}

func newSubscription(filter *topicFilter, handlers ...pubHandler) *subscription {
	r := &subscription{
		topicFilter: filter,
		handlers:    handlers,
	}
	return r
}

type subscription struct {
	*topicFilter

	handlers []pubHandler
}

func (r *subscription) String() string {
	return r.Filter()
}

// gomerge src: topicfilter.go

func mustParseTopicFilter(v string) *topicFilter {
	re, err := parseTopicFilter(v)
	if err != nil {
		panic(err.Error())
	}
	return re
}

func parseTopicFilter(v string) (*topicFilter, error) {
	if len(v) == 0 {
		return nil, fmt.Errorf("empty filter")
	}
	if i := strings.Index(v, "#"); i >= 0 && i < len(v)-1 {
		// i.e. /a/#/b
		return nil, fmt.Errorf("%q # not allowed there", v)
	}

	// build regexp
	var expr string
	if v == "#" {
		expr = "^(.*)$"
	} else {
		expr = strings.ReplaceAll(v, "+", `([\w\s]+)`)
		expr = strings.ReplaceAll(expr, "/#", `(.*)`)
		expr = "^" + expr + "$"
	}
	re, err := regexp.Compile(expr)
	if err != nil {
		return nil, err
	}

	tf := &topicFilter{
		re:     re,
		filter: v,
	}
	return tf, nil
}

// topicFilter is used to match topic names as specified in [4.7 Topic
// Names and Topic Filters]
//
// [4.7 Topic Names and Topic Filters]: https://docs.oasis-open.org/mqtt/mqtt/v5.0/os/mqtt-v5.0-os.html#_Toc3901241
type topicFilter struct {
	re     *regexp.Regexp
	filter string
}

// Match topic name and return any wildcard words.
func (r *topicFilter) Match(name string) ([]string, bool) {
	res := r.re.FindAllStringSubmatch(name, -1)
	if len(res) == 0 {
		return nil, false
	}
	// skip the entire match, ie. the first element
	return res[0][1:], true
}

func (r *topicFilter) Filter() string {
	return r.filter
}
