package tt

import (
	. "context"
	"errors"
	"net"
	"testing"
	"time"
)

func TestServer(t *testing.T) {
	{ // accepts connections
		ctx, cancel := WithCancel(Background())
		defer cancel()
		s, _ := Start(ctx, NewServer())
		<-s.Up

		conn, err := net.Dial("tcp", s.Addr().String())
		if err != nil {
			t.Fatal(err)
		}
		conn.Close()
	}
	{ // accept respects deadline
		ctx, cancel := WithCancel(Background())
		s := NewServer()
		time.AfterFunc(2*s.AcceptTimeout, cancel)
		if err := s.Run(ctx); !errors.Is(err, Canceled) {
			t.Error(err)
		}
	}
	{ // ends on listener close
		s := NewServer()
		time.AfterFunc(time.Millisecond, func() { s.Close() })
		if err := s.Run(Background()); !errors.Is(err, net.ErrClosed) {
			t.Error(err)
		}
	}
}
