package tt

import (
	"context"
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
	routes := []*Route{
		NewRoute("gopher/pink", handle),
		NewRoute("gopher/blue", NoopPub),
		NewRoute("#", handle),
	}
	r := NewRouter(routes...)

	// number of handle routes that should be triggered by below Pub
	wg.Add(2)
	ctx := context.Background()
	if err := r.Handle(ctx, mq.Pub(0, "gopher/pink", "hi")); err != nil {
		t.Error(err)
	}
	wg.Wait()
	if v := r.String(); !strings.Contains(v, "3 routes") {
		t.Error(v)
	}
}
