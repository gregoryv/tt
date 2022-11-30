package tt

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
)

type PubCmd struct {
	Server *url.URL

	Topic    string
	Payload  string
	QoS      uint8
	Timeout  time.Duration
	ClientID string

	Username string
	Password string

	Debug bool
}

func (c *PubCmd) ExtraOptions(cli *cmdline.Parser) {
	c.Server = cli.Option("-s, --server").Url("localhost:1883")
	c.Topic = cli.Option("-t, --topic").String("gopher/pink")
	c.Payload = cli.Option("-p, --payload").String("hug")
	c.QoS = cli.Option("-q, --qos").Uint8(0)
	c.Timeout = cli.Option("--timeout").Duration("1s")
	c.ClientID = cli.Option("-cid, --client-id").String("ttpub")
	c.Username = cli.Option("-u, --username").String("")
	c.Password = cli.Option("-p, --password").String("")
	c.Debug = cli.Flag("--debug")
}

func (c *PubCmd) Run(ctx context.Context) error {
	log := NewLogger()
	log.SetOutput(os.Stdout)
	log.SetLogPrefix(c.ClientID)
	log.SetDebug(c.Debug)

	log.Print("dial ", "tcp://", c.Server.String())
	conn, err := net.Dial("tcp", c.Server.String())
	if err != nil {
		return err
	}

	// use middlewares and build your in/out queues with desired
	// features
	var (
		pool = NewIDPool(100)

		transmit = CombineOut(Send(conn), log, pool)
	)

	done := make(chan struct{}, 0)
	msg := mq.Pub(c.QoS, c.Topic, c.Payload)
	handler := func(ctx context.Context, p mq.Packet) error {
		switch p := p.(type) {
		case *mq.ConnAck:
			if c.QoS == 0 {
				defer close(done)
			}
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
	in := CombineIn(handler, pool, log)
	receive := NewReceiver(conn, in)

	// kick off with a connect
	p := mq.NewConnect()
	p.SetClientID(c.ClientID)
	if c.Username != "" {
		p.SetUsername(c.Username)
		p.SetPassword([]byte(c.Password))
	}
	_ = transmit(ctx, p)

	// start handling packet flow
	ctx, cancel := context.WithTimeout(context.Background(), c.Timeout)
	defer cancel()
	_, running := Start(ctx, receive)

	select {
	case err := <-running:
		if errors.Is(err, io.EOF) {
			return fmt.Errorf("FAIL")
		}

	case <-ctx.Done():
		return ctx.Err()

	case <-done:
	}
	_ = transmit(ctx, mq.NewDisconnect())
	return nil
}
