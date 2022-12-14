package tt

import (
	"context"
	"errors"
	"io"
	"net"
	"os"
	"testing"
	"time"

	"github.com/gregoryv/mq"
)

func ExampleNewReceiver() {
	var (
		pool = NewIDPool(10)
		log  = NewLogger()
		in   = CombineIn(NoopHandler, pool, log)
	)
	_ = NewReceiver(os.Stdin, in)
	// output:
}

func TestStart(t *testing.T) {
	r := NewReceiver(&ClosedConn{}, NoopHandler)
	_, done := Start(context.Background(), r)
	select {
	case err := <-done:
		if err == nil {
			t.Fail()
		}
	}
}

func TestReceiver(t *testing.T) {
	{ // handler is called on packet from server
		conn := NewMemConn()
		called := NewCalled()
		receiver := NewReceiver(conn.Server(), called.Handler)

		go receiver.Run(context.Background())
		p := mq.NewPublish()
		p.SetTopicName("a/b")
		p.SetPayload([]byte("gopher"))
		conn.Client().Responds(p)
		<-called.Done()
	}

	{ // respects context cancellation
		// create a tcp server
		ln, err := net.Listen("tcp", ":")
		if err != nil {
			t.Fatal(err)
		}
		defer ln.Close()
		// connect to it
		conn, err := net.Dial("tcp", ln.Addr().String())
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()

		receiver := NewReceiver(conn, NoopHandler)
		receiver.readTimeout = time.Microsecond // speedup

		ctx, cancel := context.WithCancel(context.Background())
		time.AfterFunc(2*time.Microsecond, cancel)
		if err := receiver.Run(ctx); !errors.Is(err, context.Canceled) {
			t.Errorf("unexpected error: %v", err)
		}
	}

	{ // Run is stopped on closed connection
		receiver := NewReceiver(&ClosedConn{}, NoopHandler)
		err := receiver.Run(context.Background())
		if !errors.Is(err, io.EOF) {
			t.Errorf("unexpected error: %T", err)
		}
	}
}

// ----------------------------------------

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
