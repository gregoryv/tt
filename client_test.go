package tt

import (
	"context"
	"testing"
	"time"

	"github.com/gregoryv/mq"
)

func TestClient(t *testing.T) {
	server := newTestServer(t, "100ms")

	c := NewClient()
	c.SetClientID("testclient")
	c.SetServerAddr(server.URL())

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
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Millisecond)
	if err := c.Run(ctx); err != nil {
		t.Fatal(err)
	}
}

func newTestServer(t *testing.T, duration string) *Server {
	dur, err := time.ParseDuration(duration)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), dur)
	t.Cleanup(cancel)
	server, _ := Start(ctx, NewServer())
	<-server.Up
	time.AfterFunc(dur, func() { server.Close() })
	return server
}
