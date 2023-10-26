package tt

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gregoryv/mq"
	"github.com/gregoryv/tt/ttx"
)

func TestServer_DisconnectsOnMalformedSubscribe(t *testing.T) {
	conn, srvconn := net.Pipe()
	s := NewServer()
	ctx := context.Background()
	go s.Run(ctx)
	<-s.Signal() // first one is ServerUp
	go s.serveConn(ctx, srvconn)

	// initiate connect sequence
	mq.NewConnect().WriteTo(conn)
	_, _ = mq.ReadPacket(conn) // ignore ack

	{ // subscribe using malformed topic filter
		p := mq.NewSubscribe()
		p.SetPacketID(1)
		p.SetSubscriptionID(1)
		p.AddFilters(mq.NewTopicFilter("a/#/c", mq.OptQoS1))
		p.WriteTo(conn)
	}
	p, _ := mq.ReadPacket(conn)
	if p, ok := p.(*mq.Disconnect); !ok {
		t.Error("expected Disconnect got", p)
	}
}

// If a client connects without any id set the server should assign
// one in the returning ConnAck.
func TestServer_AssignsID(t *testing.T) {
	conn, srvconn := net.Pipe()
	defer conn.Close()
	s := NewServer()
	ctx := context.Background()
	go s.Run(ctx)
	<-s.Signal()
	go s.serveConn(ctx, srvconn)

	// initiate connect sequence
	mq.NewConnect().WriteTo(conn)
	p, _ := mq.ReadPacket(conn)
	if p := p.(*mq.ConnAck); p.AssignedClientID() == "" {
		t.Error("missing assigned client id")
	}
}

// If a client sends Disconnect, the server should close the network
// connection.
func TestServer_CloseConnectionOnDisconnect(t *testing.T) {
	conn, srvconn := net.Pipe()
	s := NewServer()
	ctx := context.Background()
	go s.Run(ctx)
	<-s.Signal()
	go s.serveConn(ctx, srvconn)

	{ // initiate connect sequence
		p := mq.NewConnect()
		p.WriteTo(conn)
		_, _ = mq.ReadPacket(conn) // ignore ack
	}
	{ // client sends disconnect
		mq.NewDisconnect().WriteTo(conn)
	}
	// verify that the connection is
	if _, err := mq.NewPublish().WriteTo(conn); err == nil {
		t.Error("network connection still open")
	}
}

// If server gets a malformed packet it should disconnect with the
// reason code MalformedPacket 0x81
func TestServer_DisconnectOnMalformed(t *testing.T) {
	conn, srvconn := net.Pipe()
	s := NewServer()
	ctx := context.Background()
	go s.Run(ctx)
	<-s.Signal()
	go s.serveConn(ctx, srvconn)
	{ // initiate connect sequence
		p := mq.NewConnect()
		p.WriteTo(conn)
		_, _ = mq.ReadPacket(conn) // ignore ack
	}
	{ // send malformed packet
		p := mq.NewPublish()
		p.SetQoS(mq.QoS3)
		p.WriteTo(conn)
	}
	{ // check expected disconnect packet
		p, _ := mq.ReadPacket(conn)
		if p := p.(*mq.Disconnect); p.ReasonCode() != mq.MalformedPacket {
			t.Error(p)
		}
	}
	// verify that the connection is also closed
	if _, err := mq.NewPublish().WriteTo(conn); err == nil {
		t.Error("network connection still open")
	}
}

func Test_connFeed(t *testing.T) {
	{ // accepts connections
		ctx, cancel := context.WithCancel(context.Background())
		ln, _ := net.Listen("tcp", ":")
		time.AfterFunc(3*time.Millisecond, func() {
			conn, err := net.Dial("tcp", ln.Addr().String())
			if err != nil {
				t.Fatal(err)
			}
			conn.Close()
			cancel()
		})
		f := connFeed{
			Listener:      ln,
			AcceptTimeout: time.Millisecond,
			feed:          make(chan connection, 1),
		}
		f.Run(ctx)
	}
	{ // ends on listener close
		ln, _ := net.Listen("tcp", ":")
		time.AfterFunc(time.Millisecond, func() { ln.Close() })
		f := connFeed{
			Listener:      ln,
			AcceptTimeout: time.Millisecond,
			feed:          make(chan connection, 1),
		}

		err := f.Run(context.Background())
		if !errors.Is(err, net.ErrClosed) {
			t.Error(err)
		}
	}
}

func TestSubscription_String(t *testing.T) {
	sub := mustNewSubscription("all/gophers/#", ttx.NoopPub)
	if v := sub.String(); v != "sub 0: all/gophers/#" {
		t.Errorf("unexpected subscription string %q", sub)
	}
}

func TestMustNewSubscription(t *testing.T) {
	defer catchPanic(t)
	mustNewSubscription("")
}

func catchPanic(t *testing.T) {
	if e := recover(); e == nil {
		t.Fatal("expect panic")
	}
}

func Test_router(t *testing.T) {
	var wg sync.WaitGroup
	var handle = func(_ context.Context, _ *mq.Publish) error {
		wg.Done()
		return nil
	}
	log.SetOutput(ioutil.Discard)
	r := newRouter()
	r.AddSubscriptions(
		mustNewSubscription("gopher/pink", handle),
		mustNewSubscription("gopher/blue", ttx.NoopPub),
		mustNewSubscription("#", handle),
		mustNewSubscription("#", func(_ context.Context, _ *mq.Publish) error {
			return fmt.Errorf("failed")
		}),
	)

	// number of handle routes that should be triggered by below Pub
	wg.Add(2)
	ctx := context.Background()
	if err := r.Route(ctx, mq.Pub(0, "gopher/pink", "hi")); err != nil {
		t.Error(err)
	}
	wg.Wait()
	if v := r.String(); !strings.Contains(v, "3 subscriptions") {
		t.Error(v)
	}

	// router logs errors
}

func BenchmarkRouter_10routesAllMatch(b *testing.B) {
	r := newRouter()
	for i := 0; i < 10; i++ {
		r.AddSubscriptions(mustNewSubscription("gopher/+", ttx.NoopPub))
	}

	ctx := context.Background()
	p := mq.Pub(0, "gopher/pink", "hi")
	for i := 0; i < b.N; i++ {
		if err := r.Route(ctx, p); err != nil {
			b.Error(err)
		}
	}
}

func BenchmarkRouter_10routesMiddleMatch(b *testing.B) {
	r := newRouter()
	for i := 0; i < 10; i++ {
		r.AddSubscriptions(mustNewSubscription(fmt.Sprintf("gopher/%v", i), ttx.NoopPub))
	}

	ctx := context.Background()
	p := mq.Pub(0, "gopher/5", "hi")
	for i := 0; i < b.N; i++ {
		if err := r.Route(ctx, p); err != nil {
			b.Error(err)
		}
	}
}

func BenchmarkRouter_10routesEndMatch(b *testing.B) {
	r := newRouter()
	for i := 0; i < 10; i++ {
		r.AddSubscriptions(mustNewSubscription(fmt.Sprintf("gopher/%v", i), ttx.NoopPub))
	}

	ctx := context.Background()
	p := mq.Pub(0, "gopher/9", "hi")
	for i := 0; i < b.N; i++ {
		if err := r.Route(ctx, p); err != nil {
			b.Error(err)
		}
	}
}

func ExampleTopicFilter() {
	_ = mustParseTopicFilter("/a/+/b/+/+")
}

func TestParseTopicFilter(t *testing.T) {
	// https://docs.oasis-open.org/mqtt/mqtt/v5.0/os/mqtt-v5.0-os.html#_Toc3901247
	okcases := []string{
		"#",
	}
	for _, filter := range okcases {
		err := parseTopicFilter(filter)
		if err != nil {
			t.Fatal(err)
		}
	}

	badcases := []string{
		"a/#/c",
		"#/",
		"",
	}
	for _, filter := range badcases {
		err := parseTopicFilter(filter)
		if err == nil {
			t.Fatalf("%s should fail", filter)
		}
	}
}

func TestMustParseTopicFilter_panics(t *testing.T) {
	defer catchPanic(t)
	mustParseTopicFilter("sport/ab#d")
}
