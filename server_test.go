package tt

import (
	"reflect"
	"testing"
	"time"

	"github.com/gregoryv/mq"
)

func TestServer_AddConnection(t *testing.T) {
	// Server keeps statistics of active and total connections
	s := NewServer()
	conn := dial(s)
	<-time.After(5 * time.Millisecond)
	before := s.Stat()

	conn.Close()
	<-time.After(5 * time.Millisecond)
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
	mq.NewConnect().WriteTo(conn)

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

func TestServer_DisconnectOnMalformed(t *testing.T) {
	// If server gets a malformed packet it should disconnect with the
	// reason code MalformedPacket 0x81
	s := NewServer()
	conn := dial(s)
	{
		p := mq.NewConnect()
		p.WriteTo(conn)
		_, _ = mq.ReadPacket(conn) // ignore ack
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
	if _, err := mq.NewPublish().WriteTo(conn); err == nil {
		t.Error("still connected")
	}
}
