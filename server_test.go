package tt

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/gregoryv/mq"
)

func TestServer_AddConnection(t *testing.T) {
	s := NewServer()
	conn := dial(s)
	<-time.After(10 * time.Millisecond)
	before := s.Stat()

	mq.NewDisconnect().WriteTo(conn)

	<-time.After(10 * time.Millisecond)
	after := s.Stat()

	if reflect.DeepEqual(before, after) {
		t.Error("stats are equal")
	}
}

func TestServer_AssignsID(t *testing.T) {
	// If a client connects without any id set the server should
	// assign one in the returning ConnAck.
	s := NewServer()
	conn := dial(s)
	{
		p := mq.NewConnect()
		p.WriteTo(conn)
	}
	{
		p, _ := mq.ReadPacket(conn)
		switch p := p.(type) {
		case *mq.ConnAck:
			if p.AssignedClientID() == "" {
				t.Error("missing assigned client id")
			}
		default:
			t.Fatal(p)
		}
	}
}

func TestServer_DisconnectOnMalformed(t *testing.T) {
	// If server gets a malformed packet it should disconnect with the
	// reason code MalformedPacket 0x81
	s := NewServer()
	conn := dial(s)
	{
		p := mq.NewConnect()
		p.WriteTo(conn)
		_, _ = mq.ReadPacket(conn)
	}
	{
		p := mq.NewPublish()
		p.SetQoS(mq.QoS3) // malformed
		p.WriteTo(conn)
	}
	{
		p, _ := mq.ReadPacket(conn)
		switch p := p.(type) {
		case *mq.Disconnect:
			if p.ReasonCode() != mq.MalformedPacket {
				t.Error(p)
			}
		default:
			t.Fatal(p)
		}
	}
	<-time.After(10 * time.Millisecond)
	if s := s.Stat(); s.ConnActive != 0 {
		t.Error(s)
	}
}

func xTestServer_createHandlers(t *testing.T) {
	conn := NewMemConn().Server()
	in, _ := NewServer().createHandlers(conn)
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
