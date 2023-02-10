// Package ttx provides test types
package ttx

import (
	"context"

	"github.com/gregoryv/mq"
)

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
