package tt

import (
	"context"
	"log"
	"testing"

	"github.com/gregoryv/mq"
	"github.com/gregoryv/tt/spec"
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
