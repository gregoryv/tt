package tt

import (
	"context"
	"testing"
	"time"

	"github.com/gregoryv/mq"
	"github.com/gregoryv/tt/ttx"
)

func TestIDPool(t *testing.T) {
	pool := NewIDPool(5) // 1 .. 5
	In := pool.In(ttx.NoopHandler)
	Out := pool.Out(ttx.NoopHandler)
	ctx := context.Background()

	// publish with QoS == 0
	p := mq.Pub(0, "a/b", "gopher")
	Out(ctx, p)
	if v := p.PacketID(); v != 0 {
		t.Error(p)
	}

	// packets which should get id set
	p1 := mq.Pub(1, "a/b", "gopher")
	Out(ctx, p1)

	p2 := mq.NewSubscribe()
	Out(ctx, p2)

	p3 := mq.NewUnsubscribe()
	Out(ctx, p3)

	// check that id's are different
	if p1.PacketID() == p2.PacketID() || p2.PacketID() == p3.PacketID() {
		t.Error("same id was reassigned though not reused")
	}

	// reuse IDs from these packets
	a1 := mq.NewPubAck()
	a1.SetPacketID(p1.PacketID())
	In(ctx, a1)

	a2 := mq.NewSubAck()
	a2.SetPacketID(p2.PacketID())
	In(ctx, a2)

	a3 := mq.NewUnsubAck()
	a3.SetPacketID(p3.PacketID())
	In(ctx, a3)

	// check the internal state here, this is highly dependent on the
	// steps above
	if len(pool.values) != cap(pool.values) {
		t.Error("not all ids where reused")
	}
}

func TestIDPool_nextTimeout(t *testing.T) {
	pool := NewIDPool(1) // 1 .. 5
	ctx, _ := context.WithTimeout(context.Background(), time.Millisecond)
	pool.next(ctx)
	if v := pool.next(ctx); v != 0 {
		t.Error("expect 0 id when pool is cancelled", v)
	}
}

func TestIDPool_reuse(t *testing.T) {
	pool := NewIDPool(3)
	if v := pool.reuse(99); v != 0 {
		t.Error("pool.reuse accepted value > max")
	}
	if v := pool.reuse(2); v != 0 {
		t.Error("pool.reuse accepted value that hasn't been used")
	}
}
