package tt

import (
	"context"
	"errors"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/gregoryv/mq"
	"github.com/gregoryv/tt/ttx"
)

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
			feed:          make(chan Connection, 1),
		}
		f.Run(ctx)
	}
	{ // ends on listener close
		ln, _ := net.Listen("tcp", ":")
		time.AfterFunc(time.Millisecond, func() { ln.Close() })
		f := connFeed{
			Listener:      ln,
			AcceptTimeout: time.Millisecond,
			feed:          make(chan Connection, 1),
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
	mustParseTopicFilter("sport/") // ok

	defer catchPanic(t)
	mustParseTopicFilter("sport/ab#d")
}

// ----------------------------------------

// mustNewSubscription panics on bad filter
func mustNewSubscription(filter string, handlers ...pubHandler) *subscription {
	err := parseTopicFilter(filter)
	if err != nil {
		panic(err.Error())
	}
	sub := newSubscription(handlers...)
	sub.addTopicFilter(filter)
	return sub
}
