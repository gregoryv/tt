// Package ttx provides test types
package ttx

import (
	"context"
	"github.com/gregoryv/mq"
	"fmt"
	"net"
)

func NoopHandler(_ context.Context, _ mq.Packet) error { return nil }
func NoopPub(_ context.Context, _ *mq.Publish) error   { return nil }



func NewCalled() *Called {
	return &Called{
		c: make(chan struct{}, 0),
	}
}

type Called struct {
	c chan struct{}
}

func (c *Called) Handler(_ context.Context, _ mq.ControlPacket) error {
	defer func() {
		// close may panic, just ignore it
		_ = recover()
	}()
	close(c.c)
	return nil
}

func (c *Called) Done() <-chan struct{} {
	return c.c
}


type ClosedConn struct{}

func (c *ClosedConn) Read(_ []byte) (int, error) {
	return 0, &net.OpError{Op: "read", Err: fmt.Errorf("ttx closed conn")}
}

func (c *ClosedConn) Write(_ []byte) (int, error) {
	return 0, &net.OpError{Op: "write", Err: fmt.Errorf("ttx closed conn")}
}
