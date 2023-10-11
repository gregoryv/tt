package mix

import (
	"context"

	"github.com/gregoryv/mq"
)

func ExampleFmux() {
	mx := NewFmux()
	fn := noop
	mx.Handle("sport/tennis/player1/#", fn)
	// output:
}

func noop(_ context.Context, _ *mq.Publish) error {
	return nil
}
