package tt

import (
	"context"
	"net"
	"testing"

	"github.com/gregoryv/mq"
)

// Server accepts subscribe followed by unsubscribe.
func TestServer_SubscribeThenUnsubscribe(t *testing.T) {
	ctx := context.Background()
	conn, _ := setupClientServer(ctx, t)

	{ // initiate connect sequence
		mq.NewConnect().WriteTo(conn)
		// ignore ack
		_, _ = mq.ReadPacket(conn)
	}
	{ // subscribe using malformed topic filter
		p := mq.NewSubscribe()
		p.SetPacketID(1)
		p.SetSubscriptionID(1)
		p.AddFilters(mq.NewTopicFilter("a/b/#", mq.OptQoS1))
		p.WriteTo(conn)
	}
	{ // verify ack
		p, _ := mq.ReadPacket(conn)
		if _, ok := p.(*mq.SubAck); !ok {
			t.Errorf("expected SubAck got %T", p)
		}
	}
	{ // unsubscribe
		p := mq.NewUnsubscribe()
		p.SetPacketID(1)
		p.AddFilter("a/b/#")
		p.WriteTo(conn)
	}
	{ // verify ack
		p, _ := mq.ReadPacket(conn)
		if _, ok := p.(*mq.UnsubAck); !ok {
			t.Errorf("expected UnsubAck got %T", p)
		}
	}
}

// Server sends a disconnect when a client sends a subscribe
// containing a malformed topic filter.
func TestServer_DisconnectsOnMalformedSubscribe(t *testing.T) {
	ctx := context.Background()
	conn, _ := setupClientServer(ctx, t)

	{ // initiate connect sequence
		mq.NewConnect().WriteTo(conn)
		// ignore ack
		_, _ = mq.ReadPacket(conn)
	}
	{ // subscribe using malformed topic filter
		p := mq.NewSubscribe()
		p.SetPacketID(1)
		p.SetSubscriptionID(1)
		p.AddFilters(mq.NewTopicFilter("a/#/c", mq.OptQoS1))
		p.WriteTo(conn)
	}
	// verify we recieved a disconnect packet
	p, _ := mq.ReadPacket(conn)
	if p, ok := p.(*mq.Disconnect); !ok {
		t.Error("expected Disconnect got", p)
	}
}

// If a client connects without any id set the server should assign
// one in the returning ConnAck.
func TestServer_AssignsID(t *testing.T) {
	ctx := context.Background()
	conn, _ := setupClientServer(ctx, t)

	// initiate connect sequence without client id
	mq.NewConnect().WriteTo(conn)

	// verify the acc contains assigned client id
	p, err := mq.ReadPacket(conn)
	if p := p.(*mq.ConnAck); p.AssignedClientID() == "" {
		t.Error("missing assigned client id", err)
	}
}

// If a client sends Disconnect, the server should close the network
// connection.
func TestServer_CloseConnectionOnDisconnect(t *testing.T) {
	ctx := context.Background()
	conn, _ := setupClientServer(ctx, t)

	// initiate connect sequence
	mq.NewConnect().WriteTo(conn)
	// ignore ack
	_, _ = mq.ReadPacket(conn)

	// client sends disconnect
	mq.NewDisconnect().WriteTo(conn)

	// verify the connection is closed
	if _, err := mq.NewPublish().WriteTo(conn); err == nil {
		t.Error("network Connection still open")
	}
}

// If server gets a malformed packet it should disconnect with the
// reason code MalformedPacket 0x81
func TestServer_DisconnectOnMalformed(t *testing.T) {
	ctx := context.Background()
	conn, _ := setupClientServer(ctx, t)

	{ // initiate connect sequence
		mq.NewConnect().WriteTo(conn)
		// ignore ack
		_, _ = mq.ReadPacket(conn)
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
	// verify the connection is closed
	if _, err := mq.NewPublish().WriteTo(conn); err == nil {
		t.Error("network Connection still open")
	}
}

// Publish with exceeding QoS results in disconnect.
func TestServer_DisconnectOnQoSExceed(t *testing.T) {
	ctx := context.Background()
	conn, _ := setupClientServer(ctx, t)

	{ // initiate connect sequence
		mq.NewConnect().WriteTo(conn)
		// ignore ack
		_, _ = mq.ReadPacket(conn)
	}
	{ // send publish with exceeding QoS
		p := mq.NewPublish()
		p.SetPacketID(1)
		p.SetQoS(2)
		p.SetTopicName("hello")
		p.WriteTo(conn)
	}
	{ // check expected disconnect packet
		p, _ := mq.ReadPacket(conn)
		if p := p.(*mq.Disconnect); p.ReasonCode() != mq.QoSNotSupported {
			t.Error(p)
		}
	}
	// verify the connection is closed
	if _, err := mq.NewPublish().WriteTo(conn); err == nil {
		t.Error("network Connection still open")
	}
}

func setupClientServer(ctx context.Context, t *testing.T) (conn, srvconn net.Conn) {
	s := NewServer()
	ctx, cancel := context.WithCancel(ctx)
	t.Cleanup(cancel)
	go s.Run(ctx)
	<-s.Events()

	conn, srvconn = net.Pipe()
	go s.serveConn(ctx, srvconn)
	return
}
