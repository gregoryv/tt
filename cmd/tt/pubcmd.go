package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"time"

	"github.com/gregoryv/cmdline"
	"github.com/gregoryv/mq"
	"github.com/gregoryv/tt"
)

type PubCmd struct {
	server *url.URL

	topic    string
	payload  string
	qos      uint8
	timeout  time.Duration
	clientID string

	username string
	password string

	debug bool
}

func (c *PubCmd) ExtraOptions(cli *cmdline.Parser) {
	c.server = cli.Option("-s, --server").Url("tcp://localhost:1883")
	c.clientID = cli.Option("-c, --client-id").String("ttpub")
	c.topic = cli.Option("-t, --topic-name").String("gopher/pink")
	c.payload = cli.Option("-p, --payload").String("hug")
	c.qos = cli.Option("-q, --qos").Uint8(0)
	c.timeout = cli.Option("    --timeout").Duration("1s")
	c.username = cli.Option("-u, --username").String("")
	c.password = cli.Option("-p, --password").String("")
}

func (c *PubCmd) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	if c.timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, c.timeout)
	}

	var pubErr error // wip I don't like this solution

	client := &tt.Client{
		Server:      c.server,
		Debug:       c.debug,
		MaxPacketID: 10,
		Logger:      log.New(os.Stderr, c.clientID+" ", log.Flags()),

		OnPacket: func(ctx context.Context, client *tt.Client, p mq.Packet) {
			switch p := p.(type) {
			case *mq.ConnAck:

				switch p.ReasonCode() {
				case mq.Success: // we've connected successfully
					m := mq.Pub(c.qos, c.topic, c.payload)
					if err := client.Send(ctx, m); err != nil {
						pubErr = err
						cancel()
					}
					if c.qos == 0 {
						_ = client.Send(ctx, mq.NewDisconnect())
					}

				default:
					pubErr = fmt.Errorf(p.ReasonString())
					cancel()
				}

			case *mq.PubAck:
				_ = client.Send(ctx, mq.NewDisconnect())

			default:
				pubErr = fmt.Errorf("received unexpected packet")
				cancel()
			}
		},

		OnEvent: func(ctx context.Context, client *tt.Client, e tt.Event) {
			switch e {
			case tt.EventClientUp:
				// connect
				p := mq.NewConnect()
				p.SetClientID(c.clientID)
				p.SetCleanStart(true)
				if c.username != "" {
					p.SetUsername(c.username)
					p.SetPassword([]byte(c.password))
				}
				_ = client.Send(ctx, p)
			}
		},
	}

	if err := client.Run(ctx); err != nil {
		if _, ok := err.(*net.OpError); ok {
			return err
		}
	}
	return pubErr
}
