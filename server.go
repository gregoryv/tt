package tt

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gregoryv/mq"
	"github.com/gregoryv/tt/arn"
	"github.com/gregoryv/tt/event"
)

// NewServer returns a server ready to run. Configure any settings
// before calling Run.
func NewServer() *Server {
	return &Server{
		app:      make(chan interface{}, 1),
		router:   newRouter(),
		stat:     newServerStats(),
		incoming: make(chan connection, 1),
	}
}

type Server struct {
	// where server listens for connections
	binds []*Bind

	// before initial connect packet
	connectTimeout time.Duration

	debug bool
	log   *log.Logger

	// routes publish packets to subscribing clients
	router *router

	// statistics
	stat *serverStats

	// sync server initial setup
	startup sync.Once

	// application server events, see Server.Signal()
	app chan interface{}

	// listeners feed new connections here
	incoming chan connection
}

// AddBind which to listen on for connections, defaults to
// tcp://localhost:, ie. random port on localhost.
func (s *Server) AddBind(b *Bind) {
	s.binds = append(s.binds, b)
}

// Timeout for the initial connect packet from a client before
// disconnecting, default 200ms.
func (s *Server) SetConnectTimeout(v time.Duration) {
	s.connectTimeout = v
}

// SetDebug increases log information, default false.
func (s *Server) SetDebug(v bool) {
	s.debug = v
}

// SetLogger to use for this server, defaults to no logging.
func (s *Server) SetLogger(v *log.Logger) {
	s.log = v
}

// Events returns a channel used by server to inform the application
// layer of events. E.g [event.ServerStop]
func (s *Server) Events() <-chan interface{} {
	return s.app
}

// Run the server. Use [Server.Signal] to listen for progress.
func (s *Server) Run(ctx context.Context) {
	s.startup.Do(s.setDefaults)

	if err := s.startConnectionFeeds(ctx); err != nil {
		s.app <- event.ServerStop{err}
		return
	}

	s.app <- event.ServerUp(0)
	for conn := range s.incoming {
		go s.serveConn(ctx, conn)
	}
	s.app <- event.ServerStop{nil}
}

func (s *Server) setDefaults() {
	// no logging if logger not set
	if s.log == nil {
		s.log = log.New(ioutil.Discard, "", 0)
	}
	if s.debug {
		s.log.SetFlags(s.log.Flags() | log.Lshortfile)
	}
	if s.connectTimeout == 0 {
		s.connectTimeout = 200 * time.Millisecond
	}
	if len(s.binds) == 0 {
		s.AddBind(&Bind{
			URL:           "tcp://localhost:",
			AcceptTimeout: "500ms",
		})
	}
}

// startConnectionFeeds creates listeners for configured binds
// and runs them.
func (s *Server) startConnectionFeeds(ctx context.Context) error {
	// Each bind feeds the server with connections
	for _, b := range s.binds {
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
			feed:          s.incoming,
			Listener:      ln,
			AcceptTimeout: t,
		}
		go f.Run(ctx)
	}
	return nil
}

// ----------------------------------------

// Bind holds server listening settings
type Bind struct {
	// eg. tcp://localhost[:port]
	URL string

	// eg. 500ms
	AcceptTimeout string
}

// ----------------------------------------

type connFeed struct {
	// Listener to watch
	net.Listener

	AcceptTimeout time.Duration

	// serveConn handles new remote connections
	feed chan<- connection
}

// Run blocks until context is cancelled or accepting a connection
// fails. Accepting new connection can only be interrupted if listener
// has SetDeadline method.
func (f *connFeed) Run(ctx context.Context) error {
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
		f.feed <- conn
	}
}

// ----------------------------------------

// newRouter returns a router for handling the given subscriptions.
func newRouter() *router {
	return &router{
		rut: arn.NewTree(),
		log: log.New(log.Writer(), "router ", log.Flags()),
	}
}

type router struct {
	m   sync.Mutex
	rut *arn.Tree
	log *log.Logger
}

func (r *router) String() string {
	return plural(len(r.rut.Leafs()), "subscription")
}

func (r *router) AddSubscriptions(v ...*subscription) {
	r.m.Lock()
	defer r.m.Unlock()
	for _, s := range v {
		for _, f := range s.filters {
			n := r.rut.AddFilter(f)
			if n.Value == nil {
				n.Value = v
			} else {
				n.Value = append(n.Value.([]*subscription), s)
			}
		}
	}
}

// Route routes mq.Publish packets by topic name.
func (r *router) Route(ctx context.Context, p mq.Packet) error {
	switch p := p.(type) {
	case *mq.Publish:
		// optimization opportunity by pooling a set of results
		var result []*arn.Node
		r.rut.Match(&result, p.TopicName())
		for _, n := range result {
			for _, s := range n.Value.([]*subscription) {
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

// ----------------------------------------

// MustNewSubscription panics on bad filter
func mustNewSubscription(filter string, handlers ...pubHandler) *subscription {
	err := parseTopicFilter(filter)
	if err != nil {
		panic(err.Error())
	}
	sub := newSubscription(handlers...)
	sub.addTopicFilter(filter)
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

	filters []string
	// todo multiple clients can share a subscription

	handlers []pubHandler
}

func (r *subscription) String() string {
	switch len(r.filters) {
	case 0:
		return fmt.Sprintf("sub %v", r.subscriptionID)
	case 1:
		return fmt.Sprintf("sub %v: %s", r.subscriptionID, r.filters[0])
	default:
		return fmt.Sprintf("sub %v: %s...", r.subscriptionID, r.filters[0])
	}

}

func (s *subscription) addTopicFilter(f string) {
	s.filters = append(s.filters, f)
}

// ----------------------------------------

func mustParseTopicFilter(v string) string {
	err := parseTopicFilter(v)
	if err != nil {
		panic(err.Error())
	}
	return v
}

// https://docs.oasis-open.org/mqtt/mqtt/v5.0/os/mqtt-v5.0-os.html#_Toc3901247
func parseTopicFilter(v string) error {
	if len(v) == 0 {
		return fmt.Errorf("empty filter")
	}
	if i := strings.Index(v, "#"); i >= 0 && i < len(v)-1 {
		// i.e. /a/#/b
		return fmt.Errorf("%q # not allowed there", v)
	}

	return nil
}
