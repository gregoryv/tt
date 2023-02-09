package ttsrv

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/gregoryv/mq"
	"github.com/gregoryv/testnet"
)

func TestServer_AddConnection(t *testing.T) {
	// Server keeps statistics of active and total connections
	conn, srvconn := testnet.Dial("tcp", "someserver:1234")
	s := NewServer()
	go s.AddConnection(context.Background(), srvconn)
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
	conn, srvconn := testnet.Dial("tcp", "someserver:1234")
	s := NewServer()
	go s.AddConnection(context.Background(), srvconn)
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
	conn, srvconn := testnet.Dial("tcp", "someserver:1234")
	s := NewServer()
	go s.AddConnection(context.Background(), srvconn)
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
