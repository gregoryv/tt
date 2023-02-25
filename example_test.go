package tt_test

import (
	"context"

	"github.com/gregoryv/mq"
	"github.com/gregoryv/tt"
)

// Example shows a simple client for connect, publish a QoS 0 and
// disconnect.
func Example_client() {
	client := &tt.Client{
		Server: "tcp://localhost:1883",
	}

	ctx := context.Background()
	packets, events := client.Start(ctx)

	for {
		select {
		case p := <-packets:
			onPacket(ctx, client, p)

		case e := <-events:
			onEvent(ctx, client, e)

		case <-ctx.Done():
			return
		}
	}
}

func onPacket(ctx context.Context, c *tt.Client, p mq.Packet) {
	switch p := p.(type) {
	case *mq.ConnAck:

		switch p.ReasonCode() {
		case mq.Success:
			p := mq.Pub(0, "gopher/happy", "yes")
			_ = c.Send(ctx, p)
		}
	}
}

func onEvent(ctx context.Context, c *tt.Client, e tt.Event) {
	switch e {
	case tt.EventClientUp:
		_ = c.Send(ctx, mq.NewConnect())
	}
}
