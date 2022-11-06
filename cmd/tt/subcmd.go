package main

import (
	"context"
	"fmt"
	"net"
	"net/url"

	"github.com/gregoryv/cmdline"
	"github.com/gregoryv/mq"
	"github.com/gregoryv/tt"
)

type SubCmd struct {
	server      *url.URL
	topicFilter string
}

func (c *SubCmd) ExtraOptions(cli *cmdline.Parser) {
	c.server = cli.Option("-s, --server").Url("localhost:1883")
	c.topicFilter = cli.Option("-f, --topic-filter").String("#")
}

func (c *SubCmd) Run(ctx context.Context) error {
	conn, err := net.Dial("tcp", c.server.String())
	if err != nil {
		return err
	}

	// use middlewares and build your in/out queues with desired
	// features
	var (
		pool   = tt.NewIDPool(100)
		logger = tt.NewLogger(tt.LevelInfo)

		out     = pool.Out(logger.Out(tt.Send(conn)))
		handler tt.Handler
	)

	handler = func(ctx context.Context, p mq.Packet) error {
		switch p := p.(type) {
		case *mq.ConnAck:
			sub := mq.NewSubscribe()
			sub.AddFilter(c.topicFilter, mq.OptNL)
			return out(ctx, sub)

		case *mq.Publish:
			if p.PacketID() > 0 {
				ack := mq.NewPubAck()
				ack.SetPacketID(p.PacketID())
				return out(ctx, ack)
			}
			fmt.Println("PAYLOAD", string(p.Payload()))
		default:

		}
		return nil
	}

	// start handling packet flow
	in := logger.In(pool.In(handler))
	r := tt.NewReceiver(in, conn)
	_, done := tt.Start(context.Background(), r)

	// kick off with a connect
	p := mq.NewConnect()
	p.SetClientID("ttsub")
	_ = out(ctx, p)

	<-done

	// todo handle ctrl+c with gracefule disconnect
	return nil
}
