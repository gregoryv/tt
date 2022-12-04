package tt

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"
)

func TestListener(t *testing.T) {
	{ // fails to run on bad bind
		s := NewListener()
		binds := []string{
			"tcp://:-1883",
			"jibberish",
		}
		for _, b := range binds {
			s.Bind = b
			if err := s.Run(context.Background()); err == nil {
				t.Errorf("Bind %s should fail", b)
			}
		}
	}
	{ // accepts connections
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		s, _ := Start(ctx, NewListener())
		<-s.Up

		conn, err := net.Dial("tcp", s.Addr().String())
		if err != nil {
			t.Fatal(err)
		}
		conn.Close()
	}
	{ // accept respects deadline
		ctx, cancel := context.WithCancel(context.Background())
		s := NewListener()
		time.AfterFunc(2*s.AcceptTimeout, cancel)
		if err := s.Run(ctx); !errors.Is(err, context.Canceled) {
			t.Error(err)
		}
	}
	{ // ends on listener close
		s := NewListener()
		time.AfterFunc(time.Millisecond, func() { s.Close() })
		err := s.Run(context.Background())
		if !errors.Is(err, net.ErrClosed) {
			t.Error(err)
		}
	}
}
