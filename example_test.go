package tt_test

import (
	"context"
	"log"

	"github.com/gregoryv/mq"
	"github.com/gregoryv/tt"
)

// Example shows a simple client for connect, publish a QoS 0 and
// disconnect.
func Example_client() {
	server := tt.NewServer()
	ctx := context.Background()
	go server.Run(ctx)

	client := &tt.Client{
		Server: server.Binds[0].URL,
	}

	app := func(ctx context.Context, p mq.Packet) error {
		switch p := p.(type) {
		case *mq.ConnAck:
			switch p.ReasonCode() {
			case mq.Success: // we've connected successfully
				// publish a message
				p := mq.Pub(0, "gopher/happy", "yes")
				return client.Send(ctx, p)
			}
		}
		return nil
	}

	onEvent := func(ctx context.Context, e tt.Event) {
		switch e {
		case tt.EventRunning:
			_ = client.Send(ctx, mq.NewConnect())
		}
	}

	if err := client.Run(ctx, app, onEvent); err != nil {
		log.Fatal(err)
	}
}
