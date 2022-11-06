package tt

import (
	. "context"
	"errors"
	"net"
	"testing"
	"time"
)

func TestServer(t *testing.T) {
	{ // Accepts connections
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
	{ // Accept respects deadline
		ctx, cancel := WithCancel(Background())
		s := NewServer()
		time.AfterFunc(2*s.AcceptTimeout, cancel)
		if err := s.Run(ctx); !errors.Is(err, Canceled) {
			t.Error(err)
		}
	}
	{ // Ends on listener close
		s := NewServer()
		time.AfterFunc(time.Millisecond, func() { s.Close() })
		if err := s.Run(Background()); !errors.Is(err, net.ErrClosed) {
			t.Error(err)
		}
	}
}
