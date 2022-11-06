package tt

import (
	"context"
	"io"
	"testing"

	"github.com/gregoryv/mq"
)

func TestSender(t *testing.T) {
	ctx := context.Background()
	p := mq.NewConnect()
	if err := Send(&ClosedConn{})(ctx, p); err == nil {
		t.Fatal("expect error")
	}
}

// ----------------------------------------

type ClosedConn struct{}

func (c *ClosedConn) Read(_ []byte) (int, error) {
	return 0, io.EOF
}

func (c *ClosedConn) Write(_ []byte) (int, error) {
	return 0, io.EOF
}
