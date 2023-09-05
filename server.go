package tt

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
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
	"github.com/gregoryv/tt/event"
)

type Server struct {
	// bind configuration where server listens for connections, empty
	// defaults to random port on localhost
	Binds []*Bind

	// client has to send the initial connect packet, default 200ms
	ConnectTimeout time.Duration

	// if nil, log output is discarded
	log *log.Logger

	// set to true for additional log information
	Debug bool

	ShowSettings bool

	// router routes incoming publish packets to subscribing clients
	router *router

	// statistics
	stat *serverStats

	once sync.Once

	// app receives server events, see Server.Signal()
	app chan interface{}
}

func (s *Server) SetLogger(v *log.Logger) {
	s.log = v
}

// Start runs the server in a separate go routine. Use [Server.Signal]
func (s *Server) Start(ctx context.Context) {
	s.once.Do(s.setDefaults)

	go func() {
		if err := s.run(ctx); err != nil {
			s.app <- event.ServerStop{err}
		}
	}()
}

func (s *Server) setDefaults() {
	// no logging if logger not set
	if s.log == nil {
		s.log = log.New(ioutil.Discard, "", 0)
	}
	if s.Debug {
		s.log.SetFlags(s.log.Flags() | log.Lshortfile)
	}
	if s.ConnectTimeout == 0 {
		s.ConnectTimeout = 200 * time.Millisecond
	}
	if len(s.Binds) == 0 {
		tcpRandom, _ := newBindConf("tcp://localhost:", "500ms")
		s.Binds = append(s.Binds, tcpRandom)
	}
	s.router = newRouter()
	s.stat = newServerStats()
	s.app = make(chan interface{}, 1)

	if s.ShowSettings {
		var buf bytes.Buffer
		if err := json.NewEncoder(&buf).Encode(s); err != nil {
			s.log.Fatal(err)
		}
		var nice bytes.Buffer
		json.Indent(&nice, buf.Bytes(), "", "  ")
		s.log.Print(nice.String())
	}
}

// Signal returns a channel used by server to inform the application
// layer of events. E.g [event.ServerStop]
func (s *Server) Signal() <-chan interface{} {
	return s.app
}

func (s *Server) run(ctx context.Context) error {

	// Each bind feeds the server with connections
	for _, b := range s.Binds {
		u, err := url.Parse(b.URL)
		if err != nil {
			return err
		}

		ln, err := net.Listen(u.Scheme, u.Host)
		if err != nil {
			return err
		}

		// log configured and actual port
		tmp := *u
		tmp.Host = ln.Addr().String()
		if u.Port() != tmp.Port() {
			s.log.Printf("bind %s (configured as %s)", tmp.String(), u.String())
		} else {
			s.log.Println("bind", u.String())
		}

		t, err := time.ParseDuration(b.AcceptTimeout)
		if err != nil {
			return err
		}

		// run the connection feed
		f := connFeed{
			serveConn:     s.serveConn,
			Listener:      ln,
			AcceptTimeout: t,
		}
		go f.Run(ctx)
	}
	s.app <- event.ServerUp(0)
	return nil
}

// serveConn handles the given remote connection. Blocks until
// receiver is done. Usually called in go routine.
func (s *Server) serveConn(ctx context.Context, conn connection) {
	s.once.Do(s.setDefaults)

	// the server tracks active connections
	addr := conn.RemoteAddr()
	a := includePort(addr.String(), s.Debug)
	connstr := fmt.Sprintf("conn %s://%s", addr.Network(), a)
	s.log.Println("new", connstr)
	s.stat.AddConn()
	defer func() {
		s.log.Println("del", connstr)
		s.stat.RemoveConn()
	}()

	var (
		m        sync.Mutex
		maxQoS   uint8 = 1 // wip support QoS 2
		maxIDLen uint  = 11

		clientID string
		shortID  string
		remote   = includePort(conn.RemoteAddr().String(), s.Debug)
	)

	// transmit packets to the connected client
	transmit := func(ctx context.Context, p mq.Packet) error {
		m.Lock()
		defer m.Unlock()

		switch p := p.(type) {
		case *mq.ConnAck:
			p.SetMaxQoS(maxQoS)
		}

		s.log.Printf("out %v -> %s@%s%s", p, shortID, remote, dump(s.Debug, p))

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

	in := func(ctx context.Context, p mq.Packet) {
		switch p := p.(type) {
		case *mq.Connect:
			// generate a client id before any logging
			clientID = p.ClientID()
			if clientID == "" {
				clientID = uuid.NewString()
			}
			shortID = trimID(clientID, maxIDLen)
		}

		s.log.Printf("in %v <- %s@%s%s", p, shortID, remote, dump(s.Debug, p))

		if p, ok := p.(interface{ WellFormed() *mq.Malformed }); ok {
			if err := p.WellFormed(); err != nil {
				d := mq.NewDisconnect()
				d.SetReasonCode(mq.MalformedPacket)
				_ = transmit(ctx, d)
			}
		}

		switch p := p.(type) {
		case *mq.PingReq:
			// 3.12.4-1 The Server MUST send a PINGRESP packet in
			// response to a PINGREQ packet
			_ = transmit(ctx, mq.NewPingResp())

		case *mq.Connect:
			a := mq.NewConnAck()
			if p.ClientID() == "" {
				a.SetAssignedClientID(clientID)
			}
			_ = transmit(ctx, a)

		case *mq.Subscribe:
			a := mq.NewSubAck()
			a.SetPacketID(p.PacketID())
			sub := newSubscription(func(ctx context.Context, p *mq.Publish) error {
				return transmit(ctx, p)
			})
			sub.subscriptionID = p.SubscriptionID()

			// check all filters
			for _, f := range p.Filters() {
				tf, err := parseTopicFilter(f.Filter())
				if err != nil {
					p := mq.NewDisconnect()
					p.SetReasonCode(mq.MalformedPacket)
					_ = transmit(ctx, p)
					return
				}
				sub.addTopicFilter(tf)

				// Subscribe.WellFormed fails if for any reason,
				// though here we want to set a reason code for each
				// filter.  3.9.3 SUBACK Payload
				a.AddReasonCode(mq.Success)
			}
			s.router.Handle(sub)
			_ = transmit(ctx, a)

		case *mq.Publish:
			// Disconnect any attempts to publish exceeding qos.
			// Specified in section 3.3.1.2 QoS
			if p.QoS() > maxQoS {
				d := mq.NewDisconnect()
				d.SetReasonCode(mq.QoSNotSupported)
				_ = transmit(ctx, d)
			}

			switch p.QoS() {
			case 0:
				_ = s.router.Route(ctx, p)
			case 1:
				ack := mq.NewPubAck()
				ack.SetPacketID(p.PacketID())
				_ = s.router.Route(ctx, p)
				_ = transmit(ctx, ack)

			case 2: // wip implement server support for QoS 2

			}

		case *mq.Disconnect:
			_ = conn.Close()
		}
	}

	// ignore error here, the connection is done
	_ = newReceiver(in, conn).Run(ctx)
	// wip remove subscriptions
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

type connection interface {
	io.ReadWriteCloser
	RemoteAddr() net.Addr
}

// gomerge src: bindconf.go

func newBindConf(uri, acceptTimeout string) (*Bind, error) {
	return &Bind{
		URL:           uri,
		AcceptTimeout: acceptTimeout,
	}, nil
}

// Bind holds server listening settings
type Bind struct {
	// eg. tcp://localhost:
	URL string

	// eg. 500ms
	AcceptTimeout string
}

type connFeed struct {
	// Listener to watch
	net.Listener

	AcceptTimeout time.Duration

	// serveConn handles new remote connections
	serveConn func(context.Context, connection)
}

// Run blocks until context is cancelled or accepting a connection
// fails. Accepting new connection can only be interrupted if listener
// has SetDeadline method.
func (f *connFeed) Run(ctx context.Context) error {
	if f.serveConn == nil {
		panic("connFeed.serveConn is nil")
	}
	l := f.Listener

loop:
	for {
		if err := ctx.Err(); err != nil {
			return nil
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
		go f.serveConn(ctx, conn)
	}
}

// ----------------------------------------

// NewRouter returns a router for handling the given subscriptions.
func newRouter() *router {
	return &router{
		subs: make([]*subscription, 0),
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

func (r *router) Handle(v ...*subscription) {
	r.subs = append(r.subs, v...)
}

// wip remove route when client disconnects

// Route routes mq.Publish packets by topic name.
func (r *router) Route(ctx context.Context, p mq.Packet) error {
	switch p := p.(type) {
	case *mq.Publish:
		// naive implementation looping over each route, improve at
		// some point
		for _, s := range r.subs {
			for _, f := range s.filters {
				if _, ok := f.Match(p.TopicName()); ok {
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
	}
	return ctx.Err()
}

func plural(v int, word string) string {
	if v > 1 {
		word = word + "s"
	}
	return fmt.Sprintf("%v %s", v, word)
}

// ----------------------------------------

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
	sub := newSubscription(handlers...)
	sub.addTopicFilter(tf)
	return sub
}

func newSubscription(handlers ...pubHandler) *subscription {
	r := &subscription{
		handlers: handlers,
	}
	return r
}

type subscription struct {
	subscriptionID int

	filters []*topicFilter // todo multiple filters for one subscription and
	// multiple clients can share a subscription

	handlers []pubHandler
}

func (r *subscription) String() string {
	switch len(r.filters) {
	case 0:
		return fmt.Sprintf("sub %v", r.subscriptionID)
	case 1:
		return fmt.Sprintf("sub %v: %s", r.subscriptionID, r.filters[0].filter)
	default:
		return fmt.Sprintf("sub %v: %s...", r.subscriptionID, r.filters[0].filter)
	}

}

func (s *subscription) addTopicFilter(f *topicFilter) {
	s.filters = append(s.filters, f)
}

// ----------------------------------------

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
