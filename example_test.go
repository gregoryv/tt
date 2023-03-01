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
	ctx := context.Background()
	client.Start(ctx)

	// v is either an packet or a event type
	for v := range client.Signal() {
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
			// do some clean up maybe
		}
	}
}

// Example shows how to run the provided server.
func Example_server() {
	var srv tt.Server
	ctx, cancel := context.WithCancel(context.Background())
	srv.Start(ctx)

	for {
		var v interface{}
		select { // wait for signal from server or
		case v = <-srv.Signal():
		case <-ctx.Done():
			return
		}

		switch v.(type) {
		case event.ServerStop:
			cancel()
		}
	}
}
