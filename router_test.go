package tt

import (
	"context"
	"fmt"
	"log"
	"testing"

	"github.com/gregoryv/mq"
	"github.com/gregoryv/tt/spec"
	"github.com/gregoryv/tt/ttx"
)

func Test_router(t *testing.T) {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	match := func(filter, name string) bool {
		var count int
		var handle = func(_ context.Context, _ *mq.Publish) error {
			count++
			return nil
		}
		r := newRouter()
		r.AddSubscriptions(
			mustNewSubscription(filter, handle),
		)

		ctx := context.Background()
		if err := r.Route(ctx, mq.Pub(0, name, "hi")); err != nil {
			t.Error(err)
		}
		return count == 1
	}
	err := spec.VerifyFilterMatching(match)
	if err != nil {
		t.Errorf("\n%v", err)
	}
}

func BenchmarkRouter_All(b *testing.B) {
	r := newRouter()
	for i := 0; i < 10; i++ {
		r.AddSubscriptions(
			mustNewSubscription("gopher/+", ttx.NoopPub),
		)
	}

	ctx := context.Background()
	p := mq.Pub(0, "gopher/pink", "hi")
	for i := 0; i < b.N; i++ {
		if err := r.Route(ctx, p); err != nil {
			b.Error(err)
		}
	}
}

func BenchmarkRouter_Middle(b *testing.B) {
	r := newRouter()
	for i := 0; i < 10; i++ {
		filter := fmt.Sprintf("gopher/%v", i)
		r.AddSubscriptions(
			mustNewSubscription(filter, ttx.NoopPub),
		)
	}

	ctx := context.Background()
	p := mq.Pub(0, "gopher/5", "hi")
	for i := 0; i < b.N; i++ {
		if err := r.Route(ctx, p); err != nil {
			b.Error(err)
		}
	}
}

func BenchmarkRouter_End(b *testing.B) {
	r := newRouter()
	for i := 0; i < 10; i++ {
		filter := fmt.Sprintf("gopher/%v", i)
		r.AddSubscriptions(
			mustNewSubscription(filter, ttx.NoopPub),
		)
	}

	ctx := context.Background()
	p := mq.Pub(0, "gopher/9", "hi")
	for i := 0; i < b.N; i++ {
		if err := r.Route(ctx, p); err != nil {
			b.Error(err)
		}
	}
}
