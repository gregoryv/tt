package tt

import (
	"context"
	"testing"

	"github.com/gregoryv/mq"
	"github.com/gregoryv/tt/tttest"
)

func TestSender(t *testing.T) {
	ctx := context.Background()
	p := mq.NewConnect()
	if err := Send(&tttest.ClosedConn{})(ctx, p); err == nil {
		t.Fatal("expect error")
	}
}
