package tt

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"strings"
	"sync"
	"testing"

	"github.com/gregoryv/mq"
	"github.com/gregoryv/tt/ttx"
)

func Test_router(t *testing.T) {
	var wg sync.WaitGroup
	var handle = func(_ context.Context, _ *mq.Publish) error {
		wg.Done()
		return nil
	}
	log.SetOutput(ioutil.Discard)
	r := newRouter()
	r.AddSubscriptions(
		mustNewSubscription("gopher/pink", handle),
		mustNewSubscription("gopher/blue", ttx.NoopPub),
		mustNewSubscription("#", handle),
		mustNewSubscription("#", func(_ context.Context, _ *mq.Publish) error {
			return fmt.Errorf("failed")
		}),
	)

	// number of handle routes that should be triggered by below Pub
	wg.Add(2)
	ctx := context.Background()
	if err := r.Route(ctx, mq.Pub(0, "gopher/pink", "hi")); err != nil {
		t.Error(err)
	}
	wg.Wait()
	if v := r.String(); !strings.Contains(v, "3 subscriptions") {
		t.Error(v)
	}

	// router logs errors
}
