package tt_test

import (
	"context"
	"log"

	"github.com/gregoryv/mq"
	"github.com/gregoryv/tt"
	"github.com/gregoryv/tt/event"
)

// Example shows a simple client for connect, publish a QoS 0 and
// disconnect.
func Example_client() {
	client := tt.NewClient()
	client.SetServer("tcp://localhost:1883")

	ctx := context.Background()
	go client.Run(ctx)

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
	srv := tt.NewServer()
	ctx := context.Background()
	go srv.Run(ctx)

	for v := range srv.Signal() {
		switch v := v.(type) {
		case event.ServerStop:
			if v.Err != nil {
				log.Println(v.Err)
			}
		}
	}
}
