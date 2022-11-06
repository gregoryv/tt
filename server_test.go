package tt

import (
	"context"
	. "context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/gregoryv/mq"
)

func TestServer(t *testing.T) {
	{ // fails to run on bad bind
		s := NewServer()
		s.Bind = "jibberish"
		if err := s.Run(context.Background()); err == nil {
			t.Fail()
		}
	}
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

func TestServer_CreateHandlers(t *testing.T) {
	conn := Dial()
	in, _ := NewServer().CreateHandlers(conn)
	{ // accepted packets
		packets := []mq.Packet{
			mq.NewConnect(),
			mq.Pub(0, "a/b", ""),
			func() mq.Packet {
				p := mq.Pub(1, "a/b", "")
				p.SetPacketID(1)
				return p
			}(),
			func() mq.Packet {
				p := mq.Pub(2, "a/b", "")
				p.SetPacketID(11)
				return p
			}(),
			func() mq.Packet {
				p := mq.NewPubRel()
				p.SetPacketID(12)
				return p
			}(),
		}
		ctx := context.Background()
		for _, p := range packets {
			if err := in(ctx, p); err != nil {
				t.Fatal(p, err)
			}
		}
	}
	{ // denied packets
		packets := []mq.Packet{
			mq.Pub(0, "", ""), // malformed, missing topic
			mq.NewPubComp(),
			mq.NewPingReq(),
			mq.NewSubscribe(),
		}
		ctx := context.Background()
		for _, p := range packets {
			if err := in(ctx, p); err == nil {
				t.Errorf("expect incoming handler to fail on %p", p)
			}
		}
	}
}
