package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/gregoryv/cmdline"
	"github.com/gregoryv/mq"
	"github.com/gregoryv/tt"
)

type SubCmd struct {
	output      io.Writer
	clientID    string
	topicFilter string
	keepAlive   time.Duration

	*tt.Client
}

func (c *SubCmd) ExtraOptions(cli *cmdline.Parser) {
	c.Client.Server = cli.Option("-s, --server").Url("tcp://localhost:1883")
	c.clientID = cli.Option("-c, --client-id").String("ttsub")
	c.topicFilter = cli.Option("-t, --topic-filter").String("#")
	c.keepAlive = cli.Option("-k, --keep-alive", "disable with 0").Duration("10s")
}

func (c *SubCmd) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)

	client := c.Client
	client.MaxPacketID = 10
	client.Logger = log.New(os.Stderr, c.clientID+" ", log.Flags())

	client.OnPacket = func(ctx context.Context, client *tt.Client, p mq.Packet) {
		switch p := p.(type) {
		case *mq.ConnAck:

			switch p.ReasonCode() {
			case mq.Success: // we've connected successfully
				// subscribe
				s := mq.NewSubscribe()
				s.SetSubscriptionID(1)
				f := mq.NewTopicFilter(c.topicFilter, mq.OptNL)
				s.AddFilters(f)
				_ = client.Send(ctx, s)

			default:
				fmt.Fprintln(os.Stderr, p.ReasonString())
				cancel()
			}

		case *mq.Publish:
			fmt.Fprintln(c.output, "PAYLOAD", string(p.Payload()))
		}
	}

	client.OnEvent = func(ctx context.Context, client *tt.Client, e tt.Event) {
		switch e {
		case tt.EventClientUp:
			p := mq.NewConnect()
			p.SetClientID(c.clientID)
			p.SetReceiveMax(1)
			p.SetKeepAlive(uint16(c.keepAlive.Seconds()))
			_ = client.Send(ctx, p)
		}
	}

	return client.Run(ctx)
}
