package ttsrv

import (
	"context"
	"testing"

	"github.com/gregoryv/mq"
	"github.com/gregoryv/tt/ttx"
)

func TestSubscriber(t *testing.T) {
	r := NewRouter()
	called := ttx.NewCalled()
	s := NewSubscriber(r, called.Handler)

	{
		p := mq.NewSubscribe()
		p.AddFilters(mq.NewTopicFilter("a/b", mq.OptQoS1))
		s.In(ttx.NoopHandler)(context.Background(), p)
	}

	{
		p := mq.Pub(1, "a/b", "hi")
		r.Handle(context.Background(), p)
	}

	<-called.Done()
}
