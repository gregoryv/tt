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
	"github.com/gregoryv/tt/ttx"
)

type SubCmd struct {
	server      *url.URL
	topicFilter string
	clientID    string
	debug       bool
	output      io.Writer
}

func (c *SubCmd) ExtraOptions(cli *cmdline.Parser) {
	c.server = cli.Option("-s, --server").Url("localhost:1883")
	c.clientID = cli.Option("-c, --client-id").String("ttsub")
	c.topicFilter = cli.Option("-t, --topic-filter").String("#")
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

	var (
		// pool of packet ids for reuse
		pool     = tt.NewIDPool(10)
		transmit = tt.CombineOut(tt.Send(conn), log, pool)
		// FormChecker disconnects on malformed packets
		checkForm = ttsrv.NewFormChecker(transmit)

		receive = tt.NewReceiver(
			tt.CombineIn(ttx.NoopHandler, pool, checkForm, log),
			conn,
		)
	)

	{ // connect
		p := mq.NewConnect()
		p.SetClientID(c.clientID)
		if err := transmit(ctx, p); err != nil {
			return err
		}
	}

	for {
		p, err := receive.Next(ctx)
		if err != nil {
			return err
		}
		switch p := p.(type) {
		case *mq.ConnAck:
			// subscribe
			s := mq.NewSubscribe()
			s.SetSubscriptionID(1)
			f := mq.NewTopicFilter(c.topicFilter, mq.OptNL)
			s.AddFilters(f)
			if err := transmit(ctx, s); err != nil {
				return err
			}

		case *mq.Publish:
			if p.QoS() == 1 {
				ack := mq.NewPubAck()
				ack.SetPacketID(p.PacketID())
				if err := transmit(ctx, ack); err != nil {
					return err
				}
			}
			fmt.Fprintln(c.output, "PAYLOAD", string(p.Payload()))
		}
	}
}

func expect[T any](p mq.Packet, e error) (v T, err error) {
	err = e
	if err != nil {
		return
	}
	v, ok := p.(T)
	if !ok {
		err = fmt.Errorf("expected %T got: %T", v, p)
		return
	}
	return
}
