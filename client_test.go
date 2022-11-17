package tt

import (
	"context"
	"testing"
	"time"

	"github.com/gregoryv/mq"
)

func TestClient(t *testing.T) {
	c := NewClient()

	// Configure client features

	// todo replace this with our own server
	c.Dialer = func(_ context.Context) error {
		c.SetNetworkIO(Dial())
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

	dur := 10 * time.Millisecond
	ctx, _ := context.WithTimeout(context.Background(), dur)
	_ = c.Run(ctx)
}
