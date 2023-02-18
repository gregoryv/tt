package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/url"

	"github.com/gregoryv/cmdline"
	"github.com/gregoryv/mq"
	"github.com/gregoryv/tt"
	"github.com/gregoryv/tt/ttsrv"
)

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
	log.SetLogPrefix(c.clientID)
	log.SetDebug(c.debug)

	// open
	log.Print("dial tcp://", c.server.String())
	conn, err := net.Dial("tcp", c.server.String())
	if err != nil {
		return err
	}

	// wip https://docs.oasis-open.org/mqtt/mqtt/v5.0/os/mqtt-v5.0-os.html#_Keep_Alive_1
	// wip if no pingresp is received client should disconnect
	
	// pool of packet ids for reuse
	pool := tt.NewIDPool(10)

	// wip whenever a packet is transmitted the keep alive timer
	// should start over	
	transmit := tt.CombineOut(tt.Send(conn), log, pool)

	
	// FormChecker disconnects on malformed packets
	checkForm := ttsrv.NewFormChecker(transmit)

	// final handler when receiving packets
	handler := func(ctx context.Context, p mq.Packet) error {
		switch p := p.(type) {
		case *mq.ConnAck:
			// subscribe
			s := mq.NewSubscribe()
			s.SetSubscriptionID(1)
			f := mq.NewTopicFilter(c.topicFilter, mq.OptNL)
			s.AddFilters(f)
			return transmit(ctx, s)

		case *mq.Publish:
			switch p.QoS() {
			case 0: // no ack is needed
			case 1:
				ack := mq.NewPubAck()
				ack.SetPacketID(p.PacketID())
				if err := transmit(ctx, ack); err != nil {
					return err
				}
			case 2:
				return fmt.Errorf("got QoS 2: unsupported ") // todo
			}
			fmt.Fprintln(c.output, "PAYLOAD", string(p.Payload()))
		}
		return nil
	}

	receive := tt.NewReceiver(
		tt.CombineIn(handler, pool, checkForm, log),
		conn,
	)

	// connect
	p := mq.NewConnect()
	// wip p.SetKeepAlive
	p.SetClientID(c.clientID)
	p.SetReceiveMax(1) // until we have support for QoS 2
	if err := transmit(ctx, p); err != nil {
		return err
	}

	return receive.Run(ctx)
}
