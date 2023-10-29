package tt

import (
	"context"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/gregoryv/mq"
)

// serveConn handles the client connection.  Blocks until connection
// is closed or context cancelled.
func (s *Server) serveConn(ctx context.Context, conn connection) {
	// the server tracks active connections
	addr := conn.RemoteAddr()
	a := includePort(addr.String(), s.debug)
	connstr := fmt.Sprintf("conn %s://%s", addr.Network(), a)
	s.log.Println("new", connstr)
	s.stat.AddConn()
	defer func() {
		s.log.Println("del", connstr)
		s.stat.RemoveConn()
		// todo handle closed connection
	}()

	var (
		m        sync.Mutex
		maxQoS   uint8 = 1 // todo support QoS 2
		maxIDLen uint  = 11

		clientID string
		shortID  string
		remote   = includePort(conn.RemoteAddr().String(), s.debug)
	)

	// transmit packets to the connected client
	transmit := func(ctx context.Context, p mq.Packet) error {
		m.Lock()
		defer m.Unlock()

		switch p := p.(type) {
		case *mq.ConnAck:
			p.SetMaxQoS(maxQoS)
		}

		s.log.Printf("out %v -> %s@%s%s", p, shortID, remote, dump(s.debug, p))

		if _, err := p.WriteTo(conn); err != nil {
			return err
		}

		switch p.(type) {
		case *mq.Disconnect:
			// close connection after Disconnect is send
			conn.Close()
		}
		return nil
	}

	in := func(ctx context.Context, p mq.Packet) {
		switch p := p.(type) {
		case *mq.Connect:
			// generate a client id before any logging
			clientID = p.ClientID()
			if clientID == "" {
				clientID = uuid.NewString()
			}
			shortID = trimID(clientID, maxIDLen)
		}

		s.log.Printf("in %v <- %s@%s%s", p, shortID, remote, dump(s.debug, p))

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
				a.SetAssignedClientID(clientID)
			}
			_ = transmit(ctx, a)

		case *mq.Subscribe:
			a := mq.NewSubAck()
			a.SetPacketID(p.PacketID())
			sub := newSubscription(func(ctx context.Context, p *mq.Publish) error {
				return transmit(ctx, p)
			})
			sub.subscriptionID = p.SubscriptionID()

			// check all filters
			for _, f := range p.Filters() {
				filter := f.Filter()
				err := parseTopicFilter(filter)
				if err != nil {
					p := mq.NewDisconnect()
					p.SetReasonCode(mq.MalformedPacket)
					_ = transmit(ctx, p)
					return
				}
				sub.addTopicFilter(filter)

				// Subscribe.WellFormed fails if for any reason,
				// though here we want to set a reason code for each
				// filter.  3.9.3 SUBACK Payload
				a.AddReasonCode(mq.Success)
			}
			s.router.AddSubscriptions(sub)
			_ = transmit(ctx, a)

		case *mq.Unsubscribe:
			// check all filters
			filters := p.Filters()
			for _, filter := range filters {
				err := parseTopicFilter(filter)
				if err != nil {
					p := mq.NewDisconnect()
					p.SetReasonCode(mq.MalformedPacket)
					_ = transmit(ctx, p)
					return
				}
			}
			// wip remove subscriptions when client disconnects or unsubscribes
			//s.router.RemoveSubscription(filters)

		case *mq.Publish:
			// Disconnect any attempts to publish exceeding qos.
			// Specified in section 3.3.1.2 QoS
			if p.QoS() > maxQoS {
				d := mq.NewDisconnect()
				d.SetReasonCode(mq.QoSNotSupported)
				_ = transmit(ctx, d)
			}

			switch p.QoS() {
			case 0:
				_ = s.router.Route(ctx, p)
			case 1:
				ack := mq.NewPubAck()
				ack.SetPacketID(p.PacketID())
				_ = s.router.Route(ctx, p)
				_ = transmit(ctx, ack)

			case 2: // todo implement server support for QoS 2

			}

		case *mq.Disconnect:
			_ = conn.Close()
		}
	}

	// ignore error here, the connection is done
	_ = newReceiver(in, conn).Run(ctx)
}

func includePort(addr string, yes bool) string {
	if yes {
		return addr
	}
	if i := strings.Index(addr, ":"); i > 0 {
		return addr[:i]
	}
	return addr
}

type pubHandler func(context.Context, *mq.Publish) error

type connection interface {
	io.ReadWriteCloser
	RemoteAddr() net.Addr
}
