package main

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/gregoryv/cmdline"
	"github.com/gregoryv/mq"
	"github.com/gregoryv/tt"
)

type SubCmd struct {
	server      *url.URL
	topicFilter string
	clientID    string
	debug       bool
	output      io.Writer

	keepAlive time.Duration
}

func (c *SubCmd) ExtraOptions(cli *cmdline.Parser) {
	c.server = cli.Option("-s, --server").Url("tcp://localhost:1883")
	c.clientID = cli.Option("-c, --client-id").String("ttsub")
	c.topicFilter = cli.Option("-t, --topic-filter").String("#")
	c.keepAlive = cli.Option("    --keep-alive").Duration("10s")
}

func (c *SubCmd) Run(ctx context.Context) error {

	client := &tt.Client{
		Server:      c.server,
		ClientID:    c.clientID,
		Debug:       c.debug,
		KeepAlive:   uint16(c.keepAlive.Seconds()),
		MaxPacketID: 10,
	}

	app := func(ctx context.Context, p mq.Packet) error {
		switch p := p.(type) {
		case *mq.ConnAck:
			// subscribe
			s := mq.NewSubscribe()
			s.SetSubscriptionID(1)
			f := mq.NewTopicFilter(c.topicFilter, mq.OptNL)
			s.AddFilters(f)
			return client.Send(ctx, s)

		case *mq.Publish:
			fmt.Fprintln(c.output, "PAYLOAD", string(p.Payload()))
		}
		return nil
	}

	{ // connect
		p := mq.NewConnect()
		p.SetClientID(c.clientID)
		p.SetReceiveMax(1) // until we have support for QoS 2
		go time.AfterFunc(1*time.Millisecond, func() {
			_ = client.Send(ctx, p)
		})
	}
	return client.Run(ctx, app)
}
