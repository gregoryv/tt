package ttsrv

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gregoryv/mq"
	"github.com/gregoryv/testnet"
	"github.com/gregoryv/tt/ttx"
)

// Example shows how to run the provided server.
func Example_server() {
	s := NewServer()
	go s.Run(context.Background())

	b := s.Binds[0] // by default there is one
	fmt.Println(b.URL)

	// output:
	// tcp://localhost:
}

func TestServer_Run(t *testing.T) {
	s := NewServer()
	ctx, _ := context.WithTimeout(context.Background(), time.Millisecond)
	if err := s.Run(ctx); err != nil && !errors.Is(err, context.DeadlineExceeded) {
		t.Error(err)
	}
}

func TestServer_DisconnectsOnMalformedSubscribe(t *testing.T) {
	conn, srvconn := testnet.Dial("tcp", "someserver:1234")
	s := NewServer()
	go s.ServeConn(context.Background(), srvconn)

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
	conn, srvconn := testnet.Dial("tcp", "someserver:1234")
	defer conn.Close()
	s := NewServer()
	go s.ServeConn(context.Background(), srvconn)

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
	conn, srvconn := testnet.Dial("tcp", "someserver:1234")
	s := NewServer()
	go s.ServeConn(context.Background(), srvconn)

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
	conn, srvconn := testnet.Dial("tcp", "someserver:1234")
	s := NewServer()
	go s.ServeConn(context.Background(), srvconn)
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

// gomerge src: connfeed_test.go

func Test_connFeed(t *testing.T) {
	{ // accepts connections
		ctx, cancel := context.WithCancel(context.Background())
		f := newConnFeed()
		ln, _ := net.Listen("tcp", ":")

		time.AfterFunc(3*time.Millisecond, func() {
			conn, err := net.Dial("tcp", ln.Addr().String())
			if err != nil {
				t.Fatal(err)
			}
			conn.Close()
			cancel()
		})
		f.Listener = ln
		f.AcceptTimeout = time.Millisecond
		f.Run(ctx)
	}
	{ // ends on listener close
		f := newConnFeed()
		ln, _ := net.Listen("tcp", ":")
		time.AfterFunc(time.Millisecond, func() { ln.Close() })
		f.Listener = ln
		f.AcceptTimeout = time.Millisecond

		err := f.Run(context.Background())
		if !errors.Is(err, net.ErrClosed) {
			t.Error(err)
		}
	}
	{ // accepts default server
		ln := newConnFeed()
		ln.SetServer(NewServer())
	}
}

// gomerge src: subscription_test.go

func TestSubscription_String(t *testing.T) {
	sub := MustNewSubscription("all/gophers/#", ttx.NoopPub)
	if v := sub.String(); v != "all/gophers/#" {
		t.Errorf("unexpected subscription string %q", sub)
	}
}

func TestMustNewSubscription(t *testing.T) {
	defer catchPanic(t)
	MustNewSubscription("")
}

func catchPanic(t *testing.T) {
	if e := recover(); e == nil {
		t.Fatal("expect panic")
	}
}

// gomerge src: router_test.go

func TestRouter(t *testing.T) {
	var wg sync.WaitGroup
	var handle = func(_ context.Context, _ *mq.Publish) error {
		wg.Done()
		return nil
	}
	subs := []*Subscription{
		MustNewSubscription("gopher/pink", handle),
		MustNewSubscription("gopher/blue", ttx.NoopPub),
		MustNewSubscription("#", handle),
		MustNewSubscription("#", func(_ context.Context, _ *mq.Publish) error {
			return fmt.Errorf("failed")
		}),
	}
	r := NewRouter(subs...)

	// number of handle routes that should be triggered by below Pub
	wg.Add(2)
	ctx := context.Background()
	if err := r.Handle(ctx, mq.Pub(0, "gopher/pink", "hi")); err != nil {
		t.Error(err)
	}
	wg.Wait()
	if v := r.String(); !strings.Contains(v, "4 subscriptions") {
		t.Error(v)
	}

	// router logs errors
}

func BenchmarkRouter_10routesAllMatch(b *testing.B) {
	subs := make([]*Subscription, 10)
	for i, _ := range subs {
		subs[i] = MustNewSubscription("gopher/+", ttx.NoopPub)
	}
	r := NewRouter(subs...)

	for i := 0; i < b.N; i++ {
		ctx := context.Background()
		if err := r.Handle(ctx, mq.Pub(0, "gopher/pink", "hi")); err != nil {
			b.Error(err)
		}
	}
}

func BenchmarkRouter_10routesMiddleMatch(b *testing.B) {
	subs := make([]*Subscription, 10)
	for i, _ := range subs {
		subs[i] = MustNewSubscription(fmt.Sprintf("gopher/%v", i), ttx.NoopPub)
	}
	r := NewRouter(subs...)

	for i := 0; i < b.N; i++ {
		ctx := context.Background()
		if err := r.Handle(ctx, mq.Pub(0, "gopher/5", "hi")); err != nil {
			b.Error(err)
		}
	}
}

func BenchmarkRouter_10routesEndMatch(b *testing.B) {
	subs := make([]*Subscription, 10)
	for i, _ := range subs {
		subs[i] = MustNewSubscription(fmt.Sprintf("gopher/%v", i), ttx.NoopPub)
	}
	r := NewRouter(subs...)

	for i := 0; i < b.N; i++ {
		ctx := context.Background()
		if err := r.Handle(ctx, mq.Pub(0, "gopher/9", "hi")); err != nil {
			b.Error(err)
		}
	}
}
