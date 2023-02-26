package tt

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/gregoryv/mq"
	"github.com/gregoryv/testnet"
	"github.com/gregoryv/tt/ttx"
)

func Test_receiver(t *testing.T) {
	{ // handler is called on packet from server
		conn, srvconn := testnet.Dial("tcp", "someserver:1234")
		called := ttx.NewCalled()
		go newReceiver(called.Handler, srvconn).Run(context.Background())

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

		recv := newReceiver(nil, conn)
		recv.deadline = time.Microsecond // speedup

		ctx, cancel := context.WithCancel(context.Background())
		time.AfterFunc(2*time.Microsecond, cancel)
		if err := recv.Run(ctx); !errors.Is(err, context.Canceled) {
			t.Errorf("unexpected error: %v", err)
		}
	}

	{ // Run is stopped on closed connection
		recv := newReceiver(nil, &ttx.ClosedConn{})
		if err := recv.Run(context.Background()); err == nil {
			t.Errorf("receiver should fail once connection is closed")
		}
	}

}
