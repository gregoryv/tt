package tt

import (
	"context"
	"testing"

	"github.com/gregoryv/mq"
)

func TestClient(t *testing.T) {
	c := NewClient()

	// Configure client features
	c.Dialer = func(_ context.Context) error {
		c.SetNetworkIO(Dial())
		return nil
	}

	c.Run(context.Background())

	{ // can publish
		p := mq.NewPublish()
		p.SetTopicName("gopher/happy")
		p.SetPayload([]byte("yes"))
		c.Send(context.Background(), p)
	}
}
