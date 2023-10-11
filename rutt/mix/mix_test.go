package mix

import (
	"context"
	"fmt"

	"github.com/gregoryv/mq"
)

func ExampleFmux() {
	mx := NewFmux()
	fn := noop
	mx.Handle("sport/tennis/player1/#", fn)
	topic := "sport/tennis/player1"
	fmt.Println(mx.Route(topic))
	// output:
	// 1
}

func noop(_ context.Context, _ *mq.Publish) error {
	return nil
}
