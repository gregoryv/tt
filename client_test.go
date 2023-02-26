package tt

import (
	"context"
	"testing"
	"time"
)

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
