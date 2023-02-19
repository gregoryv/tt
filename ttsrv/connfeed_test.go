package ttsrv

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"
)

func TestConnFeed(t *testing.T) {
	{ // accepts connections
		ctx, cancel := context.WithCancel(context.Background())
		f := NewConnFeed()
		ln, _ := net.Listen("tcp", ":")

		time.AfterFunc(3*time.Millisecond, func() {
			conn, err := net.Dial("tcp", ln.Addr().String())
			if err != nil {
				t.Fatal(err)
			}
			conn.Close()
			cancel()
		})

		f.Run(ctx, ln)
	}
	{ // ends on listener close
		f := NewConnFeed()
		ln, _ := net.Listen("tcp", ":")
		time.AfterFunc(time.Millisecond, func() { ln.Close() })

		err := f.Run(context.Background(), ln)
		if !errors.Is(err, net.ErrClosed) {
			t.Error(err)
		}
	}
	{ // accepts default server
		ln := NewConnFeed()
		ln.SetServer(NewServer())
	}
}
