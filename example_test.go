package tt_test

import (
	"context"
	"log"
	"net"
	"time"

	"github.com/gregoryv/mq"
	"github.com/gregoryv/tt"
)

// Example shows a simple client for connect, publish a QoS 0 and
// disconnect.
func Example_client() {
	// standin for a network connection
	conn := tt.Dial()
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	transmit := tt.NewTransmitter(tt.Send(conn))
	handler := func(ctx context.Context, p mq.Packet) error {
		switch p := p.(type) {
		case *mq.ConnAck:

			switch p.ReasonCode() {
			case mq.Success: // we've connected successfully
				// publish a message
				m := mq.Pub(0, "gopher/happy", "yes")
				if err := transmit(ctx, m); err != nil {
					log.Print(err)
				}
				// disconnect
				defer cancel()
				return transmit(ctx, mq.NewDisconnect())
			}

			// handle refused connection
			return nil
		}
		return nil
	}

	// connect
	if err := transmit(ctx, mq.NewConnect()); err != nil {
		log.Fatal(err)
	}

	// start receiving packets
	receive := tt.NewReceiver(conn, handler)
	if err := receive.Run(ctx); err != nil {
		log.Print(err)
	}
}

// Example shows how to run the provided server.
func Example_server() {
	srv := tt.NewServer()
	go srv.Run(context.Background())

	// then use
	conn, err := net.Dial(srv.Addr().Network(), srv.Addr().String())
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	_ = conn // use it in your client
}
