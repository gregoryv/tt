package tt

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/gregoryv/mq"
	"github.com/gregoryv/tt/ttx"
)

func Test_receiver(t *testing.T) {
	{ // handlerFunc is called on packet from server
		conn, srvconn := net.Pipe()
		called := ttx.NewCalled()
		ctx := context.Background()

		go newReceiver(called.Handler).Run(ctx, srvconn)

		p := mq.NewPublish()
		p.SetTopicName("a/b")
		p.SetPayload([]byte("gopher"))
		p.WriteTo(conn)
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

		recv := newReceiver(nil)
		recv.deadline = time.Microsecond // speedup

		ctx, cancel := context.WithCancel(context.Background())
		time.AfterFunc(2*time.Microsecond, cancel)
		if err := recv.Run(ctx, conn); !errors.Is(err, context.Canceled) {
			t.Errorf("unexpected error: %v", err)
		}
	}

	{ // Run is stopped on closed connection
		recv := newReceiver(nil)
		ctx := context.Background()
		conn := &ttx.ClosedConn{}
		if err := recv.Run(ctx, conn); err == nil {
			t.Errorf("receiver should fail once Connection is closed")
		}
	}

}
