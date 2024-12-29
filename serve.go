package tt

import (
	"context"
	"errors"
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
	log := s.log

	// the server tracks active connections
	s.stat.AddConn()
	defer s.stat.RemoveConn()
	sc := &sclient{
		maxIDLen: 11,
		remote:   conn.RemoteAddr().String(),
		log:      s.log,
		srv:      s,
		conn:     conn,
	}
	transmit := sc.transmitter()
	receive := sc.receiver(transmit)

	var err error
	var p mq.Packet
	for {
		// read control packets from connection
		err = ctx.Err()
		if err != nil {
			break
		}
		conn.SetReadDeadline(time.Now().Add(400 * time.Millisecond))

		p, err = mq.ReadPacket(conn)
		if err != nil {
			if errors.Is(err, os.ErrDeadlineExceeded) {
				continue
			}
			break
		}
		err = receive(ctx, p)
		if err != nil {
			break
		}
	}
	if err != nil {
		log.Print(err)

		if errors.Is(err, mq.ErrMissingData) {
			// This happens e.g. if a subscribe without topic filters
			// is send. It may change if mq successfully parses the
			// packet; in that case the reason code should be
			// mq.ProtocolError
			d := mq.NewDisconnect()
			d.SetReasonCode(mq.MalformedPacket)
			_ = transmit(ctx, d)
		}

		if p := sc.mqc.Will(); p != nil {
			delay := time.Duration(sc.mqc.WillDelayInterval())
			time.AfterFunc(delay*time.Second, func() {
				_ = sc.srv.router.Route(ctx, p)
			})
		}
	}

}

// transmit or receive func
type trFunc func(context.Context, mq.Packet) error

type Connection interface {
	io.ReadWriteCloser
	SetReadDeadline(time.Time) error
	RemoteAddr() net.Addr
}

// sclient represents a connected client on the server side.
type sclient struct {
	clientID string
	shortID  string

	maxIDLen uint

	// ip:port
	remote string

	log   *log.Logger
	debug bool

	// sync transmitions
	m    sync.Mutex
	conn Connection

	mqc *mq.Connect

	srv *Server
}

func (sc *sclient) transmitter() trFunc {
	return func(ctx context.Context, p mq.Packet) error {
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
}

func (sc *sclient) receiver(transmit trFunc) trFunc {
	return func(ctx context.Context, p mq.Packet) error {
		switch p := p.(type) {
		case *mq.Connect:
			// generate a client id before any logging
			clientID := p.ClientID()
			if clientID == "" {
				clientID = uuid.NewString()
			}
			sc.clientID = clientID
			sc.shortID = trimID(clientID, sc.maxIDLen)
			sc.mqc = p
		}
		sc.log.Printf(
			"%s %v%s from %s", sc.shortID, p, dump(sc.debug, p), sc.remote,
		)

		if p, ok := p.(interface{ WellFormed() *mq.Malformed }); ok {
			if err := p.WellFormed(); err != nil {
				d := mq.NewDisconnect()
				d.SetReasonCode(mq.MalformedPacket)
				_ = transmit(ctx, d)
			}
		}

		switch p := p.(type) {
		case *mq.PingReq:
			// 3.12.4-1 The Server MUST send a PINGRESP packet in
			// response to a PINGREQ packet
			_ = transmit(ctx, mq.NewPingResp())

		case *mq.Connect:
			a := mq.NewConnAck()
			if p.ClientID() == "" {
				a.SetAssignedClientID(sc.clientID)
			}
			_ = transmit(ctx, a)
			// todo respect connectTimeout

		case *mq.Subscribe:
			a := mq.NewSubAck()
			a.SetPacketID(p.PacketID())
			sub := newSubscription(transmit)
			sub.subscriptionID = p.SubscriptionID()

			// check all filters
			for _, f := range p.Filters() {
				filter := f.Filter()
				err := parseTopicFilter(filter)
				if err != nil {
					p := mq.NewDisconnect()
					p.SetReasonCode(mq.MalformedPacket)
					return transmit(ctx, p)
				}
				sub.addTopicFilter(filter)

				// Subscribe.WellFormed fails if for any reason,
				// though here we want to set a reason code for each
				// filter.  3.9.3 SUBACK Payload
				a.AddReasonCode(mq.Success)
			}
			sc.srv.router.AddSubscriptions(sub)
			_ = transmit(ctx, a)

		case *mq.Unsubscribe:
			// check all filters
			filters := p.Filters()
			for _, filter := range filters {
				err := parseTopicFilter(filter)
				if err != nil {
					p := mq.NewDisconnect()
					p.SetReasonCode(mq.MalformedPacket)
					return transmit(ctx, p)
				}
			}
			sc.srv.router.removeFilters(sc, filters)
			{
				ack := mq.NewUnsubAck()
				ack.SetPacketID(p.PacketID())
				transmit(ctx, ack)
			}

		case *mq.Publish:
			// Disconnect any attempts to publish exceeding qos.
			// Specified in section 3.3.1.2 QoS
			if p.QoS() > sc.srv.maxQoS {
				d := mq.NewDisconnect()
				d.SetReasonCode(mq.QoSNotSupported)
				return transmit(ctx, d)
			}

			if err := parseTopicName(p.TopicName()); err != nil {
				p := mq.NewDisconnect()
				p.SetReasonCode(mq.MalformedPacket)
				return transmit(ctx, p)
			}

			switch p.QoS() {
			case 0:
				_ = sc.srv.router.Route(ctx, p)
			case 1:
				ack := mq.NewPubAck()
				ack.SetPacketID(p.PacketID())
				_ = sc.srv.router.Route(ctx, p)
				_ = transmit(ctx, ack)

			case 2:
				// todo support QoS 2

			}

		case *mq.Disconnect:
			return sc.conn.Close()
		}

		return nil
	}
}
