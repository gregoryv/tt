package tt

import (
	"context"
	"fmt"
	"testing"

	"github.com/gregoryv/mq"
)

func Test_NoopPub(t *testing.T) {
	if err := NoopPub(nil, nil); err != nil {
		t.Fatal(err)
	}
}

func Example_CombineIn() {
	end := &num{0}
	in := CombineIn(end.Handle, &num{1}, &num{2})
	in(context.Background(), mq.NewPublish())
	//output:
	// 2
	// 1
	// 0
}

func Example_CombineOut() {
	end := &num{0}
	out := CombineOut(end.Handle, &num{1}, &num{2})
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
