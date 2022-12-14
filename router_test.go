package tt

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/gregoryv/mq"
)

func TestRouter(t *testing.T) {
	var wg sync.WaitGroup
	var handle = func(_ context.Context, _ *mq.Publish) error {
		wg.Done()
		return nil
	}
	subs := []*Subscription{
		MustNewSubscription("gopher/pink", handle),
		MustNewSubscription("gopher/blue", NoopPub),
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
		subs[i] = MustNewSubscription("gopher/+", NoopPub)
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
		subs[i] = MustNewSubscription(fmt.Sprintf("gopher/%v", i), NoopPub)
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
		subs[i] = MustNewSubscription(fmt.Sprintf("gopher/%v", i), NoopPub)
	}
	r := NewRouter(subs...)

	for i := 0; i < b.N; i++ {
		ctx := context.Background()
		if err := r.Handle(ctx, mq.Pub(0, "gopher/9", "hi")); err != nil {
			b.Error(err)
		}
	}
}
