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
	client := &tt.Client{
		Server: "tcp://localhost:1883",
	}

	ctx, cancel := context.WithCancel(context.Background())
	c := client.Start(ctx)

	var v interface{}

	for {
		select {
		case v = <-c:
		case <-ctx.Done():
			return
		}

		switch v := v.(type) {
		case event.ClientUp:
			_ = client.Send(ctx, mq.NewConnect())

		case *mq.ConnAck:
			switch v.ReasonCode() {
			case mq.Success:
				// do something once you are connected
				p := mq.Pub(0, "gopher/happy", "yes")
				_ = client.Send(ctx, p)
			}

		case event.ClientStop:
			cancel()
		}
	}
}
