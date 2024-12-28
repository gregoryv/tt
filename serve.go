package tt

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gregoryv/mq"
)

// serve handles the client connection.  Blocks until connection
// is closed or context cancelled.
func serve(ctx context.Context, s *Server, conn Connection) {
	// the server tracks active connections
	s.stat.AddConn()

	sc := &sclient{
		maxIDLen: 11,
		remote:   conn.RemoteAddr().String(),
		log:      s.log,
		srv:      s,
		conn:     conn,
	}

	// read control packets from connection
	var err error
	var p mq.Packet
	for {
		err = ctx.Err()
		if err != nil {
			break
		}
		deadline := time.Now().Add(400 * time.Millisecond)
		conn.SetReadDeadline(deadline)

		p, err = mq.ReadPacket(conn)
		if err != nil {
			if errors.Is(err, os.ErrDeadlineExceeded) {
				continue
			}
			break
		}
		sc.receive(ctx, p)
	}

	if p := sc.mqc.Will(); err != nil && p != nil {
		delay := time.Duration(sc.mqc.WillDelayInterval())
		time.AfterFunc(delay*time.Second, func() {
			_ = sc.srv.router.Route(ctx, p)
		})
	}
	s.stat.RemoveConn()
}

type pubHandler func(context.Context, *mq.Publish) error

type Connection interface {
	io.ReadWriteCloser
	SetReadDeadline(time.Time) error
	RemoteAddr() net.Addr
}

// sclient represents a connected client on the server side.
type sclient struct {
	clientID string
	shortID  string
	from     string // shortID@remote

	maxIDLen uint

	remote string
	log    *log.Logger
	debug  bool

	// sync transmitions
	m    sync.Mutex
	conn Connection

	mqc *mq.Connect

	srv *Server
}

func (sc *sclient) transmit(ctx context.Context, p mq.Packet) error {
	sc.m.Lock()
	defer sc.m.Unlock()

	switch p := p.(type) {
	case *mq.ConnAck:
		p.SetMaxQoS(sc.srv.maxQoS)
	}

	sc.log.Printf("%s %v%s", sc.shortID, p, dump(sc.debug, p))

	if _, err := p.WriteTo(sc.conn); err != nil {
		return err
	}

	switch p.(type) {
	case *mq.Disconnect:
		// close Connection after Disconnect is send
		sc.conn.Close()
	}
	return nil
}

func (sc *sclient) receive(ctx context.Context, p mq.Packet) {
	switch p := p.(type) {
	case *mq.Connect:
		// generate a client id before any logging
		clientID := p.ClientID()
		if clientID == "" {
			clientID = uuid.NewString()
		}
		sc.clientID = clientID
		sc.shortID = trimID(clientID, sc.maxIDLen)
		sc.from = fmt.Sprintf("%s@%s", sc.shortID, sc.remote)
		sc.mqc = p
	}
	sc.log.Printf("%s %v%s from %s", sc.shortID, p, dump(sc.debug, p), sc.remote)

	if p, ok := p.(interface{ WellFormed() *mq.Malformed }); ok {
		if err := p.WellFormed(); err != nil {
			d := mq.NewDisconnect()
			d.SetReasonCode(mq.MalformedPacket)
			_ = sc.transmit(ctx, d)
		}
	}

	switch p := p.(type) {
	case *mq.PingReq:
		// 3.12.4-1 The Server MUST send a PINGRESP packet in
		// response to a PINGREQ packet
		_ = sc.transmit(ctx, mq.NewPingResp())

	case *mq.Connect:
		a := mq.NewConnAck()
		if p.ClientID() == "" {
			a.SetAssignedClientID(sc.clientID)
		}
		_ = sc.transmit(ctx, a)
		// todo respect connectTimeout

	case *mq.Subscribe:
		a := mq.NewSubAck()
		a.SetPacketID(p.PacketID())
		sub := newSubscription(func(ctx context.Context, p *mq.Publish) error {
			return sc.transmit(ctx, p)
		})
		sub.subscriptionID = p.SubscriptionID()

		// check all filters
		for _, f := range p.Filters() {
			filter := f.Filter()
			err := parseTopicFilter(filter)
			if err != nil {
				p := mq.NewDisconnect()
				p.SetReasonCode(mq.MalformedPacket)
				_ = sc.transmit(ctx, p)
				return
			}
			sub.addTopicFilter(filter)

			// Subscribe.WellFormed fails if for any reason,
			// though here we want to set a reason code for each
			// filter.  3.9.3 SUBACK Payload
			a.AddReasonCode(mq.Success)
		}
		sc.srv.router.AddSubscriptions(sub)
		_ = sc.transmit(ctx, a)

	case *mq.Unsubscribe:
		// check all filters
		filters := p.Filters()
		for _, filter := range filters {
			err := parseTopicFilter(filter)
			if err != nil {
				p := mq.NewDisconnect()
				p.SetReasonCode(mq.MalformedPacket)
				_ = sc.transmit(ctx, p)
				return
			}
		}
		sc.srv.router.removeFilters(sc, filters)
		{
			ack := mq.NewUnsubAck()
			ack.SetPacketID(p.PacketID())
			sc.transmit(ctx, ack)
		}

	case *mq.Publish:
		// Disconnect any attempts to publish exceeding qos.
		// Specified in section 3.3.1.2 QoS
		if p.QoS() > sc.srv.maxQoS {
			d := mq.NewDisconnect()
			d.SetReasonCode(mq.QoSNotSupported)
			_ = sc.transmit(ctx, d)
			return
		}

		if err := parseTopicName(p.TopicName()); err != nil {
			p := mq.NewDisconnect()
			p.SetReasonCode(mq.MalformedPacket)
			_ = sc.transmit(ctx, p)
			return
		}

		switch p.QoS() {
		case 0:
			_ = sc.srv.router.Route(ctx, p)
		case 1:
			ack := mq.NewPubAck()
			ack.SetPacketID(p.PacketID())
			_ = sc.srv.router.Route(ctx, p)
			_ = sc.transmit(ctx, ack)

		case 2:
			// todo support QoS 2

		}

	case *mq.Disconnect:
		_ = sc.conn.Close()
	}
}
