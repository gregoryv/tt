package mix

import (
	"context"
	"fmt"
	"testing"

	"github.com/gregoryv/mq"
)

func ExampleFmux() {
	mx := NewFmux()
	fn := noop
	mx.Handle("sport/tennis/player1/#", fn)
	topic := "sport/tennis/player1"
	pub := mq.Pub(1, topic, "... some data ...")
	fmt.Println(mx.Route(pub))
	// output:
	// 1
}

func Benchmark(b *testing.B) {
	mx := NewFmux()
	fn := noop
	mx.Handle("sport/tennis/player1/#", fn)
	topic := "sport/tennis/player1"
	pub := mq.Pub(1, topic, "... some data ...")
	for i := 0; i < b.N; i++ {
		mx.Route(pub)
	}
}

func noop(_ context.Context, _ *mq.Publish) error {
	return nil
}
