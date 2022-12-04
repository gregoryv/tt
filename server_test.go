package tt

import (
	"context"
	"testing"

	"github.com/gregoryv/mq"
)

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
			// rejected with a disconnect, but handled properly
			func() mq.Packet {
				p := mq.Pub(2, "a/b", "")
				p.SetPacketID(11)
				return p
			}(),
		}
		ctx := context.Background()
		for _, p := range packets {
			if err := in(ctx, p); err != nil {
				t.Fatal(err)
			}
		}
	}
	{ // denied packets
		packets := []mq.Packet{
			mq.Pub(0, "", ""), // malformed, missing topic
			mq.NewSubscribe(),
		}
		ctx := context.Background()
		for _, p := range packets {
			if err := in(ctx, p); err == nil {
				t.Logf("on %v", p)
				t.Errorf("expect incoming handler to fail")
			}
		}
	}
}
