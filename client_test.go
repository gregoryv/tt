package tt_test

import (
	"context"
	"log"

	"github.com/gregoryv/mq"
	"github.com/gregoryv/tt"
)

// Example shows a simple client for connect, publish QoS 0 and
// disconnect.
func Example_client() {
	// standin for a network connection
	conn := tt.Dial()
	transmit := tt.NewTransmitter(tt.Send(conn))
	handler := func(ctx context.Context, p mq.Packet) error {
		switch p.(type) {
		case *mq.ConnAck:
			// publish
			m := mq.Pub(0, "gopher/happy", "yes")
			if err := transmit(ctx, m); err != nil {
				log.Print(err)
			}
			// disconnect
			return transmit(ctx, mq.NewDisconnect())
		}
		return nil
	}

	// connect
	ctx := context.Background()
	if err := transmit(ctx, mq.NewConnect()); err != nil {
		log.Fatal(err)
	}

	// start receiving packets
	receive := tt.NewReceiver(conn, handler)
	if err := receive.Run(ctx); err != nil {
		log.Print(err)
	}
}
