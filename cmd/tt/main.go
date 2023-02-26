// Command tt is a mqtt pub/sub client and broker
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/gregoryv/cmdline"
	"github.com/gregoryv/mq"
	"github.com/gregoryv/tt"
	"github.com/gregoryv/tt/event"
)

func main() {
	// allows cmdline options to configure log flags
	log.SetFlags(0)

	var (
		cli    = cmdline.NewBasicParser()
		shared = opts{
			Debug:        cli.Flag("-d, --debug"),
			LogTimestamp: cli.Flag("-T, --log-timestamp"),
			ShowSettings: cli.Flag("-S, --show-settings"),
			Timeout:      cli.Option("--timeout", "0 means never").Duration("0"),
		}

		commands = cli.Group("Commands", "COMMAND")
		_        = commands.New("pub", &PubCmd{shared: shared})
		_        = commands.New("sub", &SubCmd{shared: shared})
		_        = commands.New("srv", &SrvCmd{shared: shared})
		cmd      = commands.Selected()
	)
	u := cli.Usage()
	u.Preface("mqtt-v5 client and broker by Gregory Vinčić")
	cli.Parse()

	if shared.LogTimestamp {
		log.SetFlags(log.LstdFlags)
	}

	// Run command in the background so we can interrupt it
	ctx, cancel := context.WithCancel(context.Background())
	if v := shared.Timeout; v > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), v)
	}
	// handle interrupt gracefully
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	go func() {
		s := <-c
		fmt.Fprintln(os.Stderr, s)
		switch s {
		case os.Kill:
			os.Exit(1)
		case os.Interrupt:
			cancel()
			return
		}
	}()
	err := cmd.(Runner).Run(ctx)
	if err != nil && !errors.Is(err, context.Canceled) {
		log.Fatal(err)
	}
}

type opts struct {
	Debug        bool
	LogTimestamp bool
	ShowSettings bool
	Timeout      time.Duration
}

type Runner interface {
	Run(context.Context) error
}

type PubCmd struct {
	shared opts

	topic   string
	payload string
	qos     uint8

	clientID string
	username string
	password string

	server *url.URL
}

func (c *PubCmd) ExtraOptions(cli *cmdline.Parser) {
	c.server = cli.Option("-s, --server").Url("tcp://localhost:1883")
	c.clientID = cli.Option("-c, --client-id").String("ttpub")
	c.topic = cli.Option("-t, --topic-name").String("gopher/pink")
	c.payload = cli.Option("-p, --payload").String("hug")
	c.qos = cli.Option("-q, --qos").Uint8(0)
	c.username = cli.Option("-u, --username").String("")
	c.password = cli.Option("-p, --password").String("")
}

func (c *PubCmd) Run(ctx context.Context) error {
	client := &tt.Client{
		Server:       c.server.String(),
		Debug:        c.shared.Debug,
		ShowSettings: c.shared.ShowSettings,
		MaxPacketID:  10,
		Logger:       log.New(os.Stderr, c.clientID+" ", log.Flags()),
	}

	ctx, cancel := context.WithCancel(ctx)
	packets, events := client.Start(ctx)
	for {
		select {
		case p := <-packets:
			switch p := p.(type) {
			case *mq.ConnAck:

				switch p.ReasonCode() {
				case mq.Success: // we've connected successfully
					m := mq.Pub(c.qos, c.topic, c.payload)
					if err := client.Send(ctx, m); err != nil {
						return err
					}
					if c.qos == 0 {
						_ = client.Send(ctx, mq.NewDisconnect())
						cancel() // we are done
					}

				default:
					return fmt.Errorf(p.ReasonString())
				}

			case *mq.PubAck:
				_ = client.Send(ctx, mq.NewDisconnect())
				cancel()

			default:
				return fmt.Errorf("received unexpected packet")
			}

		case e := <-events:
			switch e.(type) {
			case event.ClientUp:
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

		case <-ctx.Done():
			return nil
		}
	}
}

type SubCmd struct {
	shared opts

	output      io.Writer
	clientID    string
	topicFilter string
	keepAlive   time.Duration

	server *url.URL
}

func (c *SubCmd) ExtraOptions(cli *cmdline.Parser) {
	c.output = os.Stdout
	c.server = cli.Option("-s, --server").Url("tcp://localhost:1883")
	c.clientID = cli.Option("-c, --client-id").String("ttsub")
	c.topicFilter = cli.Option("-t, --topic-filter").String("#")
	c.keepAlive = cli.Option("-k, --keep-alive", "disable with 0").Duration("10s")
}

func (c *SubCmd) Run(ctx context.Context) error {

	client := &tt.Client{
		Server:       c.server.String(),
		Debug:        c.shared.Debug,
		ShowSettings: c.shared.ShowSettings,
		MaxPacketID:  10,
		Logger:       log.New(os.Stderr, c.clientID+" ", log.Flags()),
	}

	packets, events := client.Start(ctx)
	for {
		select {
		case p := <-packets:
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
					return fmt.Errorf(p.ReasonString())
				}

			case *mq.Publish:
				fmt.Fprintln(c.output, "PAYLOAD", string(p.Payload()))
			}

		case e := <-events:
			switch e.(type) {
			case event.ClientUp:
				p := mq.NewConnect()
				p.SetClientID(c.clientID)
				p.SetReceiveMax(1)
				p.SetKeepAlive(uint16(c.keepAlive.Seconds()))
				_ = client.Send(ctx, p)

			case event.ClientDown:
				return nil
			}

		case <-ctx.Done():
			return nil
		}
	}
}

type SrvCmd struct {
	shared opts
	tt.Bind
	ConnectTimeout time.Duration
}

func (c *SrvCmd) ExtraOptions(cli *cmdline.Parser) {
	c.Bind.URL = cli.Option("-b, --bind-tcp, $TT_BIND_TCP").Url("tcp://localhost:").String()
	c.Bind.AcceptTimeout = cli.Option("-a, --accept-timeout").Duration("500ms").String()
	c.ConnectTimeout = cli.Option("--connect-timeout").Duration("200ms")
}

func (c *SrvCmd) Run(ctx context.Context) error {
	s := &tt.Server{
		Debug:        c.shared.Debug,
		ShowSettings: c.shared.ShowSettings,

		Logger:         log.New(os.Stderr, "ttsrv ", log.Flags()),
		ConnectTimeout: c.ConnectTimeout,
		Binds:          []*tt.Bind{&c.Bind},
	}
	return s.Run(ctx)
}
