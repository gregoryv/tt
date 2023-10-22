package tt

import (
	"context"
	"testing"
	"time"

	"github.com/gregoryv/mq"
	"github.com/gregoryv/tt/event"
)

func TestClient(t *testing.T) {
	client := NewClient()
	client.SetServer("tcp://localhost:1883")

	ctx, _ := context.WithTimeout(context.Background(), time.Millisecond)
	client.Start(ctx)

	for v := range client.Signal() {
		switch v := v.(type) {
		case event.ClientUp:
			_ = client.Send(ctx, mq.NewConnect())

		case event.ClientConnect:
			// do something once you are connected
			p := mq.Pub(0, "gopher/happy", "yes")
			_ = client.Send(ctx, p)

		case *mq.Publish:
			_ = v // do something the received packet

		case event.ClientStop:
			// do some clean up maybe
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
