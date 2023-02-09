package ttsrv

import (
	"context"
	"io/ioutil"
	"testing"

	"github.com/gregoryv/mq"
	"github.com/gregoryv/tt"
	"github.com/gregoryv/tt/tttest"
)

func TestQualitySupport(t *testing.T) {
	f := NewQualitySupport(tt.Send(ioutil.Discard))
	ctx := context.Background()

	{ // ok pub
		p := mq.Pub(0, "a/b", "hi")
		if err := f.In(tt.NoopHandler)(ctx, p); err != nil {
			t.Fatal(err)
		}
	}
	{ // pub with unsupported qos
		p := mq.Pub(1, "a/b", "hi")
		called := tttest.NewCalled()
		if err := f.In(called.Handler)(ctx, p); err != nil {
			t.Fatal(err)
		}
		select {
		case <-called.Done():
			t.Error("was called")
		default:
		}
	}
	{ // outgoing ack
		p := mq.NewConnAck()
		if err := f.Out(tt.NoopHandler)(ctx, p); err != nil {
			t.Fatal(err)
		}
		if v := p.MaxQoS(); v != 0 {
			t.Errorf("max qos set to %v", v)
		}
	}

}
