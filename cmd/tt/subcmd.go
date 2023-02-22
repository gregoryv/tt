package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/url"

	"github.com/gregoryv/cmdline"
	"github.com/gregoryv/mq"
	"github.com/gregoryv/tt"
	"github.com/gregoryv/tt/ttsrv"
)

type SubCmd struct {
	server      *url.URL
	topicFilter string
	clientID    string
	debug       bool
	output      io.Writer

	keepAlive *tt.KeepAlive
}

func (c *SubCmd) ExtraOptions(cli *cmdline.Parser) {
	c.server = cli.Option("-s, --server").Url("localhost:1883")
	c.clientID = cli.Option("-c, --client-id").String("ttsub")
	c.topicFilter = cli.Option("-t, --topic-filter").String("#")
	c.keepAlive = tt.NewKeepAlive(
		cli.Option("    --keep-alive").Duration("10s"),
	)
}

func (c *SubCmd) Run(ctx context.Context) error {
	// use middlewares and build your in/out queues with desired
	// features
	log := tt.NewLogger()
	log.SetLogPrefix(c.clientID)
	log.SetDebug(c.debug)

	// open
	log.Print("dial tcp://", c.server.String())
	conn, err := net.Dial("tcp", c.server.String())
	if err != nil {
		return err
	}

	// pool of packet ids for reuse
	pool := tt.NewIDPool(10)

	transmit := tt.Combine(
		tt.Send(conn),
		log.Out,
		c.keepAlive.Out,
		pool.Out,
	)

	// set transmitter after as the middleware should be part of the
	// transmitter
	c.keepAlive.SetTransmitter(transmit)

	// wip decouple FormChecker disconnects on malformed packets
	checkForm := ttsrv.NewFormChecker(transmit)

	onConnAck := func(next tt.Handler) tt.Handler {
		return func(ctx context.Context, p mq.Packet) error {
			switch p.(type) {
			case *mq.ConnAck:
				// subscribe
				s := mq.NewSubscribe()
				s.SetSubscriptionID(1)
				f := mq.NewTopicFilter(c.topicFilter, mq.OptNL)
				s.AddFilters(f)
				return transmit(ctx, s)
			}
			return next(ctx, p)
		}
	}

	// final handler when receiving packets
	handler := func(ctx context.Context, p mq.Packet) error {
		switch p := p.(type) {
		case *mq.Publish:
			switch p.QoS() {
			case 0: // no ack is needed
			case 1:
				ack := mq.NewPubAck()
				ack.SetPacketID(p.PacketID())
				if err := transmit(ctx, ack); err != nil {
					return err
				}
			case 2:
				return fmt.Errorf("got QoS 2: unsupported ") // todo
			}
			fmt.Fprintln(c.output, "PAYLOAD", string(p.Payload()))
		}
		return nil
	}

	receive := tt.NewReceiver(
		tt.Combine(handler,
			onConnAck, pool.In, c.keepAlive.In, checkForm.In, log.In,
		),
		conn,
	)

	// connect
	p := mq.NewConnect()
	p.SetClientID(c.clientID)
	p.SetReceiveMax(1) // until we have support for QoS 2
	if err := transmit(ctx, p); err != nil {
		return err
	}

	return receive.Run(ctx)
}
