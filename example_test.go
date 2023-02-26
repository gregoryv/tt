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
	packets, events := client.Start(ctx)

	for {
		select {
		case p := <-packets:

			switch p := p.(type) {
			case *mq.ConnAck:

				switch p.ReasonCode() {
				case mq.Success:
					// do something once you are connected
					p := mq.Pub(0, "gopher/happy", "yes")
					_ = client.Send(ctx, p)
				}
			}

		case e := <-events:

			switch e.(type) {
			case event.ClientUp:
				_ = client.Send(ctx, mq.NewConnect())
			case event.ClientDown:
				cancel()
			}

		case <-ctx.Done():
			return
		}
	}
}
