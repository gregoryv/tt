package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"

	"github.com/gregoryv/cmdline"
	"github.com/gregoryv/mq"
	"github.com/gregoryv/tt"
	"github.com/gregoryv/tt/ttsrv"
)

var subWriter io.Writer = os.Stdout

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
	log.SetOutput(os.Stderr)
	log.SetLogPrefix(c.clientID)
	log.SetDebug(c.debug)

	// open
	log.Print("dial tcp://", c.server.String())
	conn, err := net.Dial("tcp", c.server.String())
	if err != nil {
		return err
	}

	var (
		pool     = tt.NewIDPool(100)
		transmit = tt.CombineOut(tt.Send(conn), log, pool)
		// FormChecker disconnects on malformed packets
		checkForm = ttsrv.NewFormChecker(transmit)
		handler   tt.Handler
	)

	// kick off with a connect
	p := mq.NewConnect()
	p.SetClientID(c.clientID)
	if err := transmit(ctx, p); err != nil {
		return err
	}

	handler = func(ctx context.Context, p mq.Packet) error {
		switch p := p.(type) {
		case *mq.ConnAck:
			sub := mq.NewSubscribe()
			sub.SetSubscriptionID(1)
			f := mq.NewTopicFilter(c.topicFilter, mq.OptNL)
			sub.AddFilters(f)
			return transmit(ctx, sub)

		case *mq.Publish:
			if p.QoS() == 1 {
				ack := mq.NewPubAck()
				ack.SetPacketID(p.PacketID())
				_ = transmit(ctx, ack)
			}
			fmt.Fprintln(subWriter, "PAYLOAD", string(p.Payload()))
		default:

		}
		return nil
	}

	// start handling packet flow
	in := tt.CombineIn(handler, pool, checkForm, log)
	receive := tt.NewReceiver(conn, in)

	return receive.Run(ctx)
}
