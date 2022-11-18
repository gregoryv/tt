package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
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
}

func (c *PubCmd) ExtraOptions(cli *cmdline.Parser) {
	c.server = cli.Option("-s, --server").Url("localhost:1883")
	c.topic = cli.Option("-t, --topic").String("gopher/pink")
	c.payload = cli.Option("-p, --payload").String("hug")
	c.qos = cli.Option("-q, --qos").Uint8(0)
	c.timeout = cli.Option("--timeout").Duration("1s")
	c.clientID = cli.Option("-cid, --client-id").String("ttpub")
}

func (c *PubCmd) Run(ctx context.Context) error {
	conn, err := net.Dial("tcp", c.server.String())
	if err != nil {
		return err
	}

	// use middlewares and build your in/out queues with desired
	// features
	var (
		pool     = tt.NewIDPool(100)
		logger   = tt.NewLogger(tt.LevelInfo)
		transmit = tt.NewTransmitter(pool, logger, tt.Send(conn))
	)
	logger.SetLogPrefix(c.clientID)

	done := make(chan struct{}, 0)
	msg := mq.Pub(c.qos, c.topic, c.payload)
	handler := func(ctx context.Context, p mq.Packet) error {
		switch p := p.(type) {
		case *mq.ConnAck:
			return transmit(ctx, msg)

		case *mq.PubRec:
			rel := mq.NewPubRel()
			rel.SetPacketID(p.PacketID())
			return transmit(ctx, rel)

		case *mq.PubAck:
			close(done)

		case *mq.PubComp:
			close(done)

		default:
			fmt.Println("unexpected:", p)
		}
		return nil
	}
	receive := tt.NewReceiver(conn, logger, pool, handler)

	// start handling packet flow
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()
	_, running := tt.Start(ctx, receive)

	// kick off with a connect
	p := mq.NewConnect()
	p.SetClientID(c.clientID)
	_ = transmit(ctx, p)

	select {
	case err := <-running:
		if errors.Is(err, io.EOF) {
			return fmt.Errorf("FAIL")
		}

	case <-ctx.Done():
		return ctx.Err()

	case <-done:
		defer fmt.Println("ok")
	}
	_ = transmit(ctx, mq.NewDisconnect())
	return nil
}
