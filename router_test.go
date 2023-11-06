package tt

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"strings"
	"testing"

	"github.com/gregoryv/mq"
	"github.com/gregoryv/tt/ttx"
)

func Test_router(t *testing.T) {
	var count int
	var handle = func(_ context.Context, _ *mq.Publish) error {
		count++
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
	ctx := context.Background()
	if err := r.Route(ctx, mq.Pub(0, "gopher/pink", "hi")); err != nil {
		t.Error(err)
	}
	if v := r.String(); !strings.Contains(v, "4 subscriptions, 3 filters") {
		// t.Error(v) wip
	}
	if count != 2 {
		t.Fail()
	}
}
