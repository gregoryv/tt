package tt

import (
	"context"
	"errors"
	"io/ioutil"
	"log"
	"net"
	"testing"
	"time"

	"github.com/gregoryv/tt/spec"
	"github.com/gregoryv/tt/ttx"
)

func TestServer_SetConnectTimeout(t *testing.T) {
	srv := NewServer()
	srv.SetConnectTimeout(0)
}


func TestServer_SetDebugIncreasesLogging(t *testing.T) {
	srv := NewServer()
	l := log.New(ioutil.Discard, "", 0)
	srv.SetLogger(l)
	srv.SetDebug(true)
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	t.Cleanup(cancel)
	cancel()
	srv.Run(ctx)

	if l.Flags() == 0 {
		t.Error("SetDebug did not change logger flags")
	}
}

func TestServer_cancelContext(t *testing.T) {
	srv := NewServer()
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	cancel()
	srv.Run(ctx)
	// should not block
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
	defer func() {
		if e := recover(); e == nil {
			t.Fatal("expect panic")
		}
	}()
	mustNewSubscription("")
}

func ExampleTopicFilter() {
	_ = mustParseTopicFilter("/a/+/b/+/+")
}

func TestParseTopicFilter(t *testing.T) {
	impl := func(filter string) bool {
		err := parseTopicFilter(filter)
		return err == nil
	}
	err := spec.VerifyFilterFormat(impl)
	if err != nil {
		t.Error(err)
	}
}

func Test_parseTopicName(t *testing.T) {
	bad := func(name string) {
		t.Helper()
		if err := parseTopicName(name); err == nil {
			t.Errorf("topic name %q accepted, expected error", name)
		}
	}
	bad("/+")
	bad("+")
	bad("#")
}

// ----------------------------------------

// mustNewSubscription panics on bad filter
func mustNewSubscription(filter string, handlers ...pubHandler) *subscription {
	mustParseTopicFilter(filter)
	sub := newSubscription(handlers...)
	sub.addTopicFilter(filter)
	return sub
}

func mustParseTopicFilter(v string) string {
	err := parseTopicFilter(v)
	if err != nil {
		panic(err.Error())
	}
	return v
}
