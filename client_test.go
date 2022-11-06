package tt

import (
	"context"
	"testing"

	"github.com/gregoryv/mq"
)

func TestClient(t *testing.T) {
	c := NewClient()
	c.Conn = Dial()

	{ // can publish
		p := mq.NewPublish()
		p.SetTopicName("gopher/happy")
		p.SetPayload([]byte("yes"))
		c.Send(context.Background(), p)
	}

}
