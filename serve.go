package tt

import (
	"context"
	"errors"
	"io"
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
	defer conn.Close()
	remote := conn.RemoteAddr().String()
	maxIDLen := 11
	// shortID is set when connect packet is received
	var shortID string

	// protects connection from concurrent writes
	var m sync.Mutex
	transmit := func(ctx context.Context, p mq.Packet) error {
		s.log.Printf("%s %v%s", shortID, p, dump(s.debug, p))
		m.Lock()
		_, err := p.WriteTo(conn)
		m.Unlock()
		return err
	}
	// send ignores transmit errors
	send := func(ctx context.Context, p mq.Packet) {
		_ = transmit(ctx, p)
	}
	disconnect := func(code mq.ReasonCode) {
		p := mq.NewDisconnect()
		p.SetReasonCode(code)
		send(ctx, p)
	}

	// After a Network Connection is established by a Client to a
	// Server, the first packet sent from the Client to the Server
	// MUST be a CONNECT packet [MQTT-3.1.0-1].

	// start is used to honor the server.connectTimeout
	start := time.Now()
	p, err := nextPacket(ctx, conn)
	if err != nil {
		s.log.Print(err)
		if errors.Is(err, mq.ErrMissingData) {
			disconnect(mq.MalformedPacket)
		}
		return
	}

	var pkConnect *mq.Connect
	if p, ok := p.(*mq.Connect); ok {
		// set shortID before any logging
		assignID := p.ClientID() == ""
		if assignID {
			// set it on the packet, could be a local variable
			p.SetClientID(uuid.NewString())
		}
		shortID = trimID(p.ClientID(), maxIDLen)
		s.log.Printf(
			"%s %v%s from %s", shortID, p, dump(s.debug, p), remote,
		)
		if err := s.lateConnect(start); err != nil {
			disconnect(mq.MaximumConnectTime)
			return
		}
		// save the connect packet for will handling. This probably
		// needs to be saved in a session state for this client.
		pkConnect = p
		a := mq.NewConnAck()
		if s.maxQoS < 2 {
			a.SetMaxQoS(s.maxQoS)
		}
		if assignID {
			a.SetAssignedClientID(p.ClientID())
		}
		send(ctx, a)
	}
	// client is now connected in the MQTT sense

	// The Will Message MUST be published after the Network Connection
	// is subsequently closed and either the Will Delay Interval has
	// elapsed or the Session ends, unless the Will Message has been
	// deleted by the Server on receipt of a DISCONNECT packet with
	// Reason Code 0x00 (Normal disconnection) or a new Network
	// Connection for the ClientID is opened before the Will Delay
	// Interval has elapsed [MQTT-3.1.2-8]

	// pkDisconnect is only set if the client sends it
	var pkDisconnect *mq.Disconnect
	if will := pkConnect.Will(); will != nil {
		defer func() {
			if pkDisconnect != nil {
				if pkDisconnect.ReasonCode() == mq.NormalDisconnect {
					return
				}
			}
			delay := time.Duration(pkConnect.WillDelayInterval())
			time.AfterFunc(delay*time.Second, func() {
				_ = s.router.Route(ctx, will)
			})
		}()
	}

	for {
		p, err := nextPacket(ctx, conn)
		if err != nil {
			s.log.Print(err)
			if errors.Is(err, mq.ErrMissingData) {
				// This happens e.g. if a subscribe without topic
				// filters is send. It may change if mq successfully
				// parses the packet; in that case the reason code
				// should be mq.ProtocolError
				disconnect(mq.MalformedPacket)
			}
			return
		}
		s.log.Printf(
			"%s %v%s from %s", shortID, p, dump(s.debug, p), remote,
		)
		if p, ok := p.(interface{ WellFormed() *mq.Malformed }); ok {
			if err := p.WellFormed(); err != nil {
				disconnect(mq.MalformedPacket)
				return
			}
		}

		switch p := p.(type) {
		case *mq.PingReq:
			// 3.12.4-1 The Server MUST send a PINGRESP packet in
			// response to a PINGREQ packet
			send(ctx, mq.NewPingResp())

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
					disconnect(mq.MalformedPacket)
					return
				}
				sub.addTopicFilter(filter)

				// Subscribe.WellFormed fails if for any reason,
				// though here we want to set a reason code for each
				// filter.  3.9.3 SUBACK Payload
				a.AddReasonCode(mq.Success)
			}
			s.router.AddSubscriptions(sub)
			send(ctx, a)

		case *mq.Unsubscribe:
			// check all filters
			filters := p.Filters()
			if err := checkTopicFilters(filters); err != nil {
				disconnect(mq.MalformedPacket)
				return
			}
			s.router.removeFilters(filters)

			ack := mq.NewUnsubAck()
			ack.SetPacketID(p.PacketID())
			send(ctx, ack)

		case *mq.Publish:
			// If the Server included a Maximum QoS in its CONNACK
			// response to a Client and it receives a PUBLISH packet
			// with a QoS greater than this, then it uses DISCONNECT
			// with Reason Code 0x9B (QoS not supported) as described
			// in section 4.13 Handling errors.			
			if p.QoS() > s.maxQoS {
				disconnect(mq.QoSNotSupported)
				return
			}

			if err := parseTopicName(p.TopicName()); err != nil {
				disconnect(mq.MalformedPacket)
				return
			}

			switch p.QoS() {
			case 0:
				_ = s.router.Route(ctx, p)
			case 1:
				ack := mq.NewPubAck()
				ack.SetPacketID(p.PacketID())
				_ = s.router.Route(ctx, p)
				send(ctx, ack)

			case 2:
				// support QoS 2

			}

		case *mq.Disconnect:
			// needed for the defered will logic
			pkDisconnect = p
			return
		}
	}
}

func nextPacket(ctx context.Context, conn Connection) (mq.Packet, error) {
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		conn.SetReadDeadline(time.Now().Add(400 * time.Millisecond))

		p, err := mq.ReadPacket(conn)
		if err != nil {
			if !errors.Is(err, os.ErrDeadlineExceeded) {
				return nil, err
			}
			continue
		}
		return p, nil
	}
}

type Connection interface {
	io.ReadWriteCloser
	SetReadDeadline(time.Time) error
	RemoteAddr() net.Addr
}

func checkTopicFilters(filters []string) error {
	for _, filter := range filters {
		err := parseTopicFilter(filter)
		if err != nil {
			return err
		}
	}
	return nil
}

// transmit or receive func
type trFunc func(context.Context, mq.Packet) error
