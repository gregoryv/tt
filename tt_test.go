package tt

import (
	"context"
	"fmt"

	"github.com/gregoryv/mq"
)

func Example_Combine() {
	end := &num{0}
	one := &num{1}
	two := &num{2}
	out := Combine(end.Handle, one.Out, two.Out)
	out(context.Background(), mq.NewPublish())
	//output:
	// 2
	// 1
	// 0
}

type num struct {
	i int
}

func (c *num) Handle(ctx context.Context, p mq.Packet) error {
	fmt.Println(c.i)
	return nil
}

func (c *num) In(next Handler) Handler {
	return func(ctx context.Context, p mq.Packet) error {
		fmt.Println(c.i)
		return next(ctx, p)
	}
}

func (c *num) Out(next Handler) Handler {
	return func(ctx context.Context, p mq.Packet) error {
		fmt.Println(c.i)
		return next(ctx, p)
	}
}
