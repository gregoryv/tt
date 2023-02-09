package ttsrv

import (
	"context"
	"testing"

	"github.com/gregoryv/mq"
	"github.com/gregoryv/tt"
)

func Test_IDPool(t *testing.T) {
	pool := NewIDPool(5) // 1 .. 5
	ctx := context.Background()

	p := mq.Pub(1, "a/b", "gopher")
	if err := pool.In(tt.NoopHandler)(ctx, p); err != nil {
		t.Fatal(err)
	}

	if err := pool.Out(tt.NoopHandler)(ctx, p); err != nil {
		t.Fatal(err)
	}
}
