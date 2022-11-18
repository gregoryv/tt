package tt

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/gregoryv/mq"
)

func TestClient(t *testing.T) {
	dur := 10 * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), dur)
	defer cancel()
	server, _ := Start(ctx, NewServer())
	<-server.Up

	c := NewClient()
	c.SetClientID("testclient")

	// Configure client features
	c.Dialer = func(_ context.Context) error {
		conn, err := net.Dial("tcp", server.Addr().String())
		if err != nil {
			return err
		}
		c.SetNetworkIO(conn)
		return nil
	}

	c.Handler = func(_ context.Context, p mq.Packet) error {
		switch p.(type) {
		case *mq.ConnAck:
			p := mq.NewPublish()
			p.SetTopicName("gopher/happy")
			p.SetPayload([]byte("yes"))

			return c.Send(context.Background(), p)
		}
		return nil
	}

	if err := c.Run(ctx); err != nil {
		t.Fatal(err)
	}
}
