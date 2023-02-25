package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/gregoryv/cmdline"
	"github.com/gregoryv/tt"
)

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
		Logger:         log.New(os.Stderr, "ttsrv ", log.Flags()),
		ConnectTimeout: c.ConnectTimeout,
	}
	s.Binds = append(s.Binds, &c.Bind)
	return s.Run(ctx)
}
