package tt_test

import (
	"context"
	"log"
	"net/url"

	"github.com/gregoryv/mq"
	"github.com/gregoryv/tt"
)

// Example shows a simple client for connect, publish a QoS 0 and
// disconnect.
func Example_client() {
	srv, _ := url.Parse("tcp://localhost:1883")

	client := &tt.Client{
		Server: srv,

		OnPacket: func(ctx context.Context, c *tt.Client, p mq.Packet) {
			switch p := p.(type) {
			case *mq.ConnAck:

				switch p.ReasonCode() {
				case mq.Success:
					p := mq.Pub(0, "gopher/happy", "yes")
					_ = c.Send(ctx, p)
				}

			}
		},

		OnEvent: func(ctx context.Context, c *tt.Client, e tt.Event) {
			switch e {
			case tt.EventClientUp:
				_ = c.Send(ctx, mq.NewConnect())
			}
		},
	}

	if err := client.Run(context.Background()); err != nil {
		log.Fatal(err)
	}
}
