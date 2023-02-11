package tt_test

import (
	"context"
	"log"
	"time"

	"github.com/gregoryv/mq"
	"github.com/gregoryv/testnet"
	"github.com/gregoryv/tt"
)

// Example shows a simple client for connect, publish a QoS 0 and
// disconnect.
func Example_client() {
	conn, _ := testnet.Dial("tcp", "someserver:1234")

	// transmit handler for synced packet writes
	transmit := tt.Send(conn)

	receiver := tt.NewReceiver(conn,
		// handler for received packets
		func(ctx context.Context, p mq.Packet) error {
			switch p := p.(type) {
			case *mq.ConnAck:

				switch p.ReasonCode() {
				case mq.Success: // we've connected successfully
					// publish a message
					p := mq.Pub(0, "gopher/happy", "yes")
					if err := transmit(ctx, p); err != nil {
						log.Print(err)
					}
					// disconnect
					_ = transmit(ctx, mq.NewDisconnect())
					return tt.StopReceiver
				}
			}
			return nil
		},
	)

	// initiate connect sequence
	ctx, _ := context.WithTimeout(context.Background(), time.Millisecond)
	if err := transmit(ctx, mq.NewConnect()); err != nil {
		log.Fatal(err)
	}

	// start receiving packets
	if err := receiver.Run(ctx); err != nil {
		log.Print(err)
	}
}
