package tt

import (
	"context"
	"io/ioutil"
	"testing"

	"github.com/gregoryv/mq"
)

func TestQualitySupport_In(t *testing.T) {
	f := NewQualitySupport(Send(ioutil.Discard))
	f.In(NoopHandler)(context.Background(), mq.Pub(0, "a/b", "hi"))
}
