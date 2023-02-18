package tt_test

import (
	"context"
	"log"
	"net"
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

	// initiate connect sequence
	ctx, _ := context.WithTimeout(context.Background(), time.Millisecond)
	if err := transmit(ctx, mq.NewConnect()); err != nil {
		log.Fatal(err)
	}

	// define how to handle incoming packets using a receiver
	receiver := tt.NewReceiver(
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
		}, conn,
	)

	// start receiving packets
	if err := receiver.Run(ctx); err != nil {
		log.Print(err)
	}
}

func Example_publishQoS1() {
	// open
	conn, err := net.Dial("tcp", "localhost:1883")
	if err != nil {
		log.Fatal(err)
	}
	if _, err := mq.NewConnect().WriteTo(conn); err != nil {
		log.Fatal(err)
	}
	// wait for connack
	if _, err := mq.ReadPacket(conn); err != nil {
		log.Fatal(err)
	}

	p := mq.Pub(1, "a/b", "hello")
	p.SetPacketID(1)
	if _, err := p.WriteTo(conn); err != nil {
		log.Fatal(err)
	}

	if _, err := mq.ReadPacket(conn); err != nil {
		log.Fatal(err)
	}
	if _, err := mq.NewDisconnect().WriteTo(conn); err != nil {
		log.Fatal(err)
	}
	// close
	conn.Close()
}
