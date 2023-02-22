package main

import (
	"context"
	"log"
	"net/url"
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
	c.timeout = cli.Option("    --timeout").Duration("1s") // wip global ?
	c.username = cli.Option("-u, --username").String("")
	c.password = cli.Option("-p, --password").String("")
}

func (c *PubCmd) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)

	client := &tt.Client{
		Server:      c.server,
		ClientID:    c.clientID,
		Debug:       c.debug,
		KeepAlive:   uint16(10),
		MaxPacketID: 10,

		OnPacket: func(ctx context.Context, client *tt.Client, p mq.Packet) error {
			switch p := p.(type) {
			case *mq.ConnAck:

				switch p.ReasonCode() {
				case mq.Success: // we've connected successfully
					m := mq.Pub(c.qos, c.topic, c.payload)
					err := client.Send(ctx, m)
					if err != nil {
						log.Print(err)
					}
					cancel()

				default:
					log.Print(p.ReasonString())
					cancel()
				}
				return nil

			}
			return nil
		},

		OnEvent: func(ctx context.Context, client *tt.Client, e tt.Event) {
			switch e {
			case tt.EventRunning:
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

	return client.Run(ctx)
}
