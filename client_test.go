package tt

import (
	"context"
	"testing"
	"time"

	"github.com/gregoryv/mq"
	"github.com/gregoryv/tt/event"
)

func TestClientIO(t *testing.T) {
	t.Run("mosquitto", func(t *testing.T) {
		testClient(t, "tcp://localhost:1883")
	})
	t.Run("tt", func(t *testing.T) {
		testClient(t, "tcp://localhost:11883")
	})
}

func testClient(t *testing.T, server string) {
	t.Run("pub QoS 1", func(t *testing.T) {
		pubQoS1(t, server)
	})
	t.Run("connect will message", func(t *testing.T) {
		willMessage(t, server)
	})
}

func pubQoS1(t *testing.T, server string) {
	client := NewClient()
	// assuming a broker is running
	client.SetServer(server)

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	go client.Run(ctx)

	for v := range client.Events() {
		switch v := v.(type) {
		case event.ClientUp:
			p := mq.NewConnect()
			p.SetClientID("sparrow")
			if err := client.Send(ctx, p); err != nil {
				t.Fatal(err)
			}

		case event.ClientConnect:
			// do something once you are connected
			p := mq.Pub(1, "cat/alert", "outside")
			if err := client.Send(ctx, p); err != nil {
				t.Fatal(err)
			}

		case *mq.PubAck:
			p := mq.NewDisconnect()
			if err := client.Send(ctx, p); err != nil {
				t.Fatal(err)
			}
			cancel()
			return

		default:
			t.Log(v)
		}
	}
}

func willMessage(t *testing.T, server string) {
	// nest watch
	watch := NewClient()
	watch.SetServer(server)

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go watch.Run(ctx)

	expect := "sparrow left"
	ready := make(chan struct{})
	gotWill := make(chan struct{})
	go func() {
		for v := range watch.Events() {
			switch v := v.(type) {
			case event.ClientUp:
				p := mq.NewConnect()
				p.SetClientID("watch")
				if err := watch.Send(ctx, p); err != nil {
					t.Fatal(err)
				}

			case event.ClientConnect:
				p := mq.NewSubscribe()
				f := mq.NewTopicFilter("nest", mq.OptNL)
				p.AddFilters(f)
				if err := watch.Send(ctx, p); err != nil {
					t.Fatal(err)
				}
				close(ready)

			case *mq.Publish:
				if m := string(v.Payload()); m == expect {
					watch.Send(ctx, mq.NewDisconnect())
					close(gotWill)
					return
				} else {
					t.Fatalf("got: %q", m)
				}
			}
		}
	}()
	<-ready

	// the test client
	client := NewClient()
	client.SetServer(server)
	ctx, forceDisconnect := context.WithCancel(ctx)
	go client.Run(ctx)

	var willDelay time.Duration // whole seconds only
	for v := range client.Events() {
		switch v := v.(type) {
		case event.ClientUp:
			p := mq.NewConnect()
			p.SetClientID("sparrow")
			p.SetWill(mq.Pub(0, "nest", expect))
			p.SetWillDelayInterval(uint32(willDelay / time.Second))
			if err := client.Send(ctx, p); err != nil {
				t.Fatal(err)
			}

		case event.ClientConnect:
			forceDisconnect()
			delay := willDelay + 2*time.Second
			select {
			case <-time.After(delay):
				t.Fatalf("no will within %v", delay)

			case <-gotWill:
				return
			}

		default:
			t.Log(v)
		}
	}
}

func Test_iDPool_nextTimeout(t *testing.T) {
	pool := newIDPool(1) // 1 .. 5
	ctx, _ := context.WithTimeout(context.Background(), time.Millisecond)
	pool.next(ctx)
	if v, _ := pool.next(ctx); v != 0 {
		t.Error("expect 0 id when pool is cancelled", v)
	}
}

func Test_iDPool_reuse(t *testing.T) {
	pool := newIDPool(3)
	if v := pool.reuse(99); v != 0 {
		t.Error("pool.reuse accepted value > max")
	}
	if v := pool.reuse(2); v != 0 {
		t.Error("pool.reuse accepted value that hasn't been used")
	}
}
