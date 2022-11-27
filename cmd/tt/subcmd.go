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
	c.topicFilter = cli.Option("-f, --topic-filter").String("#")
	c.clientID = cli.Option("-cid, --client-id").String("ttsub")
	c.debug = cli.Flag("--debug")
}

func (c *SubCmd) Run(ctx context.Context) error {
	conn, err := net.Dial("tcp", c.server.String())
	if err != nil {
		return err
	}

	// use middlewares and build your in/out queues with desired
	// features
	var (
		pool = tt.NewIDPool(100)
		log  = tt.NewLogger()

		transmit = tt.CombineOut(tt.Send(conn), pool, log)
		handler  tt.Handler
	)
	log.SetOutput(os.Stdout)
	log.SetLogPrefix(c.clientID)
	log.SetDebug(c.debug)

	handler = func(ctx context.Context, p mq.Packet) error {
		switch p := p.(type) {
		case *mq.ConnAck:
			sub := mq.NewSubscribe()
			f := mq.NewTopicFilter(c.topicFilter, mq.OptNL)
			sub.AddFilters(f)
			return transmit(ctx, sub)

		case *mq.Publish:
			if p.PacketID() > 0 {
				ack := mq.NewPubAck()
				ack.SetPacketID(p.PacketID())
				return transmit(ctx, ack)
			}
			fmt.Fprintln(subWriter, "PAYLOAD", string(p.Payload()))
		default:

		}
		return nil
	}

	// start handling packet flow
	in := tt.CombineIn(handler, log, pool)
	receive := tt.NewReceiver(conn, in)
	_, done := tt.Start(context.Background(), receive)

	// kick off with a connect
	p := mq.NewConnect()
	p.SetClientID("ttsub")
	_ = transmit(ctx, p)

	<-done

	// todo handle ctrl+c with gracefule disconnect
	return nil
}
