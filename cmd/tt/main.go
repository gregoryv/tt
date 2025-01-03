// Command tt is a mqtt pub/sub client and broker
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	_ "net/http/pprof"
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
			Quiet:        cli.Option("-Q, --quiet", "hide log output").Bool(false),
			Timeout:      cli.Option("--timeout", "0 means never").Duration("0"),
		}

		commands = cli.Group("Commands", "COMMAND")
		_        = commands.New("pub", &PubCmd{shared: shared})
		_        = commands.New("sub", &SubCmd{shared: shared})
		_        = commands.New("srv", &SrvCmd{shared: shared})
		_        = commands.New("bench", &BenchCmd{shared: shared})
		cmd      = commands.Selected()
	)
	u := cli.Usage()
	u.Preface("mqtt-v5 client and broker by Gregory Vinčić")
	cli.Parse()

	if shared.LogTimestamp {
		log.SetFlags(log.LstdFlags)
	}
	log.SetOutput(shared.LogOutput())
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
	Quiet        bool
	Timeout      time.Duration
}

func (o *opts) LogOutput() io.Writer {
	if o.Quiet {
		return ioutil.Discard
	}
	return os.Stderr
}

type Runner interface {
	Run(context.Context) error
}

type PubCmd struct {
	shared opts

	topic   string
	payload string
	qos     uint8
	retain  bool
	count   int

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
	c.count = cli.Option("-n, --count").Int(1)
	c.retain = cli.Option("-r, --retain").Bool(false)
}

func (c *PubCmd) Run(ctx context.Context) error {
	client := tt.NewClient()
	client.SetServer(c.server.String())
	client.SetDebug(c.shared.Debug)
	client.SetMaxPacketID(10)

	client.SetLogger(
		log.New(c.shared.LogOutput(), c.clientID+" ", log.Flags()),
	)

	ctx, cancel := context.WithCancel(ctx)
	go client.Run(ctx)
	for v := range client.Events() {
		switch v := v.(type) {
		case event.ClientUp:
			p := mq.NewConnect()
			p.SetClientID(c.clientID)
			p.SetCleanStart(true)
			if c.username != "" {
				p.SetUsername(c.username)
				p.SetPassword([]byte(c.password))
			}
			_ = client.Send(ctx, p)

		case event.ClientConnect:
			for i := 0; i < c.count; i++ {
				m := mq.Pub(c.qos, c.topic, c.payload)
				m.SetRetain(c.retain)
				if err := client.Send(ctx, m); err != nil {
					return err
				}
			}
			if c.qos == 0 {
				_ = client.Send(ctx, mq.NewDisconnect())
				cancel() // we are done
			}

		case event.ClientConnectFail:
			cancel()
			return fmt.Errorf(string(v))

		case *mq.PubAck:
			_ = client.Send(ctx, mq.NewDisconnect())
			cancel()

		case *mq.PubComp:
			_ = client.Send(ctx, mq.NewDisconnect())
			cancel()

		case *mq.Disconnect:
			cancel()
			if r := v.ReasonCode(); r > 0x80 {
				return fmt.Errorf(v.String())
			}
		}
	}
	return ctx.Err()
}

type SubCmd struct {
	shared opts

	output      io.Writer
	clientID    string
	topicFilter string
	filterQoS   uint8
	keepAlive   time.Duration

	autoReconnect bool

	server *url.URL
}

func (c *SubCmd) ExtraOptions(cli *cmdline.Parser) {
	c.output = os.Stdout
	c.server = cli.Option("-s, --server").Url("tcp://localhost:1883")
	c.clientID = cli.Option("-c, --client-id").String("ttsub")
	c.topicFilter = cli.Option("-t, --topic-filter").String("#")
	c.filterQoS = cli.Option("-q, --qos", "filter quality of service").Uint8(1)
	c.keepAlive = cli.Option("-k, --keep-alive", "disable with 0").Duration("10s")
	c.autoReconnect = cli.Flag("-r, --auto-reconnect")
}

func (c *SubCmd) Run(ctx context.Context) error {
	for {
		err := c.run(ctx)
		if !c.autoReconnect {
			return err
		}
		if err != nil {
			log.Print(err)
		}
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(2 * time.Second):
		}

	}
}

func (c *SubCmd) run(ctx context.Context) error {
	client := tt.NewClient()
	client.SetServer(c.server.String())
	client.SetDebug(c.shared.Debug)
	client.SetMaxPacketID(10)
	client.SetLogger(
		log.New(c.shared.LogOutput(), c.clientID+" ", log.Flags()),
	)

	ctx, cancel := context.WithCancel(ctx)
	go client.Run(ctx)

	for v := range client.Events() {
		switch v := v.(type) {
		case event.ClientUp:
			p := mq.NewConnect()
			p.SetClientID(c.clientID)
			p.SetReceiveMax(1)
			p.SetKeepAlive(uint16(c.keepAlive.Seconds()))
			_ = client.Send(ctx, p)

		case *mq.ConnAck:
			switch v.ReasonCode() {
			case mq.Success:
				s := mq.NewSubscribe()
				s.SetSubscriptionID(1)
				opts := mq.OptNL
				if v := c.filterQoS; v < 3 {
					opts |= mq.FilterOption(v)
				}
				f := mq.NewTopicFilter(c.topicFilter, opts)
				s.AddFilters(f)
				_ = client.Send(ctx, s)
			default:
				cancel()
				return fmt.Errorf(v.ReasonString())
			}

		case *mq.Publish:
			if c.shared.Quiet {
				fmt.Fprintln(c.output, v.TopicName(), string(v.Payload()))
			} else {
				// when logging is enabled the publish packet contains
				// the topic
				fmt.Fprintln(c.output, "PAYLOAD", string(v.Payload()))
			}

		case event.ClientStop:
			cancel()
			return v.Err
		}
	}
	return ctx.Err()
}

type SrvCmd struct {
	shared opts
	tt.Bind
	profile        bool
	ConnectTimeout time.Duration
	LogPrefix      string
}

func (c *SrvCmd) ExtraOptions(cli *cmdline.Parser) {
	c.Bind.URL = cli.Option("-b, --bind-tcp, $TT_BIND_TCP").Url("tcp://localhost:").String()
	c.Bind.AcceptTimeout = cli.Option("-a, --accept-timeout").Duration("500ms").String()
	c.ConnectTimeout = cli.Option("--connect-timeout").Duration("200ms")
	c.LogPrefix = cli.Option("--log-prefix", "").String("")
	c.profile = cli.Flag("--pprof")
}

func (c *SrvCmd) Run(ctx context.Context) error {
	if c.profile {
		go func() {
			fmt.Println("pprof :9999")
			log.Println(http.ListenAndServe(":9999", nil))
		}()
	}
	srv := tt.NewServer()
	srv.SetDebug(c.shared.Debug)
	srv.SetConnectTimeout(c.ConnectTimeout)
	srv.AddBind(&c.Bind)
	if c.LogPrefix != "" {
		c.LogPrefix += " "
	}
	srv.SetLogger(
		log.New(c.shared.LogOutput(), c.LogPrefix, log.Flags()),
	)
	ctx, cancel := context.WithCancel(ctx)
	go srv.Run(ctx)

	for {
		select {
		case v := <-srv.Events():
			switch v.(type) {
			case event.ServerStop:
				cancel()
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
