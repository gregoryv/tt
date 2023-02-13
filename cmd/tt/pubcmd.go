package main

import (
	"context"
	"errors"
	"fmt"
	"io"
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
	c.server = cli.Option("-s, --server").Url("localhost:1883")
	c.topic = cli.Option("-t, --topic").String("gopher/pink")
	c.payload = cli.Option("-p, --payload").String("hug")
	c.qos = cli.Option("-q, --qos").Uint8(0)
	c.timeout = cli.Option("--timeout").Duration("1s")
	c.clientID = cli.Option("-cid, --client-id").String("ttpub")
	c.username = cli.Option("-u, --username").String("")
	c.password = cli.Option("-p, --password").String("")
	c.debug = cli.Flag("--debug")
}

func (c *PubCmd) Run(ctx context.Context) error {
	// create logger
	log := tt.NewLogger()
	log.SetOutput(os.Stdout)
	log.SetLogPrefix(c.clientID)
	log.SetDebug(c.debug)

	// create network connection
	log.Print("dial ", "tcp://", c.server.String())
	conn, err := net.Dial("tcp", c.server.String())
	if err != nil {
		return err
	}

	// limit number of concurrent packets
	pool := tt.NewIDPool(10)

	// transmit is used for every outgoing packet
	transmit := tt.CombineOut(tt.Send(conn), log, pool)

	handler := func(ctx context.Context, p mq.Packet) error {
		switch p := p.(type) {
		case *mq.ConnAck:
			// once connected send publish
			m := mq.Pub(c.qos, c.topic, c.payload)
			err := transmit(ctx, m)
			if c.qos == 0 {
				return tt.StopReceiver
			}
			return err

		case *mq.PubRec:
			rel := mq.NewPubRel()
			rel.SetPacketID(p.PacketID())
			return transmit(ctx, rel)

		case *mq.PubAck:
			return tt.StopReceiver
		case *mq.PubComp:
			return tt.StopReceiver
		case *mq.Disconnect:
			return tt.StopReceiver

		default:
			return fmt.Errorf("got unexpected packet: %v", p)
		}
		return nil
	}
	in := tt.CombineIn(handler, pool, log)
	receiver := tt.NewReceiver(conn, in)

	{ // initiate connect sequence
		p := mq.NewConnect()
		p.SetClientID(c.clientID)
		p.SetCleanStart(true)
		if c.username != "" {
			p.SetUsername(c.username)
			p.SetPassword([]byte(c.password))
		}
		_ = transmit(ctx, p)
	}

	// start receiving packets
	ctx, _ = context.WithTimeout(ctx, c.timeout)
	if err := receiver.Run(ctx); errors.Is(err, io.EOF) {
		return fmt.Errorf("FAIL")
	}

	_ = transmit(ctx, mq.NewDisconnect())
	return nil
}
