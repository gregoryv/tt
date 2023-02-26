package tt_test

import (
	"context"

	"github.com/gregoryv/mq"
	"github.com/gregoryv/tt"
	"github.com/gregoryv/tt/event"
)

// Example shows a simple client for connect, publish a QoS 0 and
// disconnect.
func Example_client() {
	client := tt.Client{
		Server: "tcp://localhost:1883",
	}

	ctx, cancel := context.WithCancel(context.Background())
	in := client.Start(ctx)

	// v is either an packet type or a event type
	var v interface{}
	for {
		select {
		case v = <-in:
		case <-ctx.Done():
			return
		}

		switch v := v.(type) {
		case event.ClientUp:
			_ = client.Send(ctx, mq.NewConnect())

		case event.ClientConnect:
			// do something once you are connected
			p := mq.Pub(0, "gopher/happy", "yes")
			_ = client.Send(ctx, p)

		case *mq.Publish:
			_ = v // do something the received packet

		case event.ClientStop:
			cancel()
		}
	}
}
