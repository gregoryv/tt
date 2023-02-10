package ttx

import (
	"context"

	"github.com/gregoryv/mq"
)

func NoopPub(_ context.Context, _ *mq.Publish) error { return nil }
