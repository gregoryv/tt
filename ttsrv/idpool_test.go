package ttsrv

import (
	"context"
	"testing"

	"github.com/gregoryv/mq"
	"github.com/gregoryv/tt/ttx"
)

func Test_IDPool(t *testing.T) {
	pool := NewIDPool(5) // 1 .. 5
	ctx := context.Background()

	p := mq.Pub(1, "a/b", "gopher")
	if err := pool.In(ttx.NoopHandler)(ctx, p); err != nil {
		t.Fatal(err)
	}

	if err := pool.Out(ttx.NoopHandler)(ctx, p); err != nil {
		t.Fatal(err)
	}
}
