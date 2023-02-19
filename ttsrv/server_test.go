package ttsrv

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/gregoryv/mq"
	"github.com/gregoryv/testnet"
)

func TestServer_Run(t *testing.T) {
	s := NewServer()
	ctx, _ := context.WithTimeout(context.Background(), time.Millisecond)
	if err := s.Run(ctx); err != nil && !errors.Is(err, context.DeadlineExceeded) {
		t.Error(err)
	}
}

func TestServer_DisconnectsOnMalformedSubscribe(t *testing.T) {
	conn, srvconn := testnet.Dial("tcp", "someserver:1234")
	s := NewServer()
	go s.ServeConn(context.Background(), srvconn)

	// initiate connect sequence
	mq.NewConnect().WriteTo(conn)
	_, _ = mq.ReadPacket(conn) // ignore ack

	{ // subscribe using malformed topic filter
		p := mq.NewSubscribe()
		p.SetPacketID(1)
		p.SetSubscriptionID(1)
		p.AddFilters(mq.NewTopicFilter("a/#/c", mq.OptQoS1))
		p.WriteTo(conn)
	}
	p, _ := mq.ReadPacket(conn)
	if p, ok := p.(*mq.Disconnect); !ok {
		t.Error("expected Disconnect got", p)
	}
}

// Server keeps statistics of active and total connections
func TestServer_AddConnection(t *testing.T) {
	conn, srvconn := testnet.Dial("tcp", "someserver:1234")
	s := NewServer()

	// server increases number of connections
	go s.ServeConn(context.Background(), srvconn)
	<-time.After(5 * time.Millisecond)
	before := s.Stat()

	// server decreases number again
	conn.Close()
	<-time.After(5 * time.Millisecond)
	after := s.Stat()

	if reflect.DeepEqual(before, after) {
		t.Error("stats are equal")
	}
}

// If a client connects without any id set the server should assign
// one in the returning ConnAck.
func TestServer_AssignsID(t *testing.T) {
	conn, srvconn := testnet.Dial("tcp", "someserver:1234")
	defer conn.Close()
	s := NewServer()
	go s.ServeConn(context.Background(), srvconn)

	// initiate connect sequence
	mq.NewConnect().WriteTo(conn)
	p, _ := mq.ReadPacket(conn)
	if p := p.(*mq.ConnAck); p.AssignedClientID() == "" {
		t.Error("missing assigned client id")
	}
}

// If a client sends Disconnect, the server should close the network
// connection.
func TestServer_CloseConnectionOnDisconnect(t *testing.T) {
	conn, srvconn := testnet.Dial("tcp", "someserver:1234")
	s := NewServer()
	go s.ServeConn(context.Background(), srvconn)

	{ // initiate connect sequence
		p := mq.NewConnect()
		p.WriteTo(conn)
		_, _ = mq.ReadPacket(conn) // ignore ack
	}
	{ // client sends disconnect
		mq.NewDisconnect().WriteTo(conn)
	}
	// verify that the connection is
	if _, err := mq.NewPublish().WriteTo(conn); err == nil {
		t.Error("network connection still open")
	}
}

// If server gets a malformed packet it should disconnect with the
// reason code MalformedPacket 0x81
func TestServer_DisconnectOnMalformed(t *testing.T) {
	conn, srvconn := testnet.Dial("tcp", "someserver:1234")
	s := NewServer()
	go s.ServeConn(context.Background(), srvconn)
	{ // initiate connect sequence
		p := mq.NewConnect()
		p.WriteTo(conn)
		_, _ = mq.ReadPacket(conn) // ignore ack
	}
	{ // send malformed packet
		p := mq.NewPublish()
		p.SetQoS(mq.QoS3)
		p.WriteTo(conn)
	}
	{ // check expected disconnect packet
		p, _ := mq.ReadPacket(conn)
		if p := p.(*mq.Disconnect); p.ReasonCode() != mq.MalformedPacket {
			t.Error(p)
		}
	}
	// verify that the connection is also closed
	if _, err := mq.NewPublish().WriteTo(conn); err == nil {
		t.Error("network connection still open")
	}
}
