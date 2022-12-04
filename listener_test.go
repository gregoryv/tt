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
		ln, _ := Start(ctx, NewListener())
		<-ln.Up

		conn, err := net.Dial("tcp", ln.Addr().String())
		if err != nil {
			t.Fatal(err)
		}
		conn.Close()
	}
	{ // accept respects deadline
		ctx, cancel := context.WithCancel(context.Background())
		ln := NewListener()
		time.AfterFunc(2*ln.AcceptTimeout, cancel)
		if err := ln.Run(ctx); !errors.Is(err, context.Canceled) {
			t.Error(err)
		}
	}
	{ // ends on listener close
		ln := NewListener()
		time.AfterFunc(time.Millisecond, func() { ln.Close() })
		err := ln.Run(context.Background())
		if !errors.Is(err, net.ErrClosed) {
			t.Error(err)
		}
	}
	{ // accepts default server
		ln := NewListener()
		ln.SetServer(NewServer())
	}
}
