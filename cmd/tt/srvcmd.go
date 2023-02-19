package main

import (
	"context"

	"github.com/gregoryv/cmdline"
	"github.com/gregoryv/tt/ttsrv"
)

type SrvCmd struct {
	binds []ttsrv.BindConf

	debug bool

	srv *ttsrv.Server
}

func (c *SrvCmd) ExtraOptions(cli *cmdline.Parser) {
	var b ttsrv.BindConf
	b.URL = cli.Option("-b, --bind-tcp, $TT_BIND_TCP").Url("tcp://localhost:1883")
	b.AcceptTimeout = cli.Option("-a, --accept-timeout").Duration("1s")
	b.Debug = c.debug
	c.binds = append(c.binds, b)

	s := ttsrv.NewServer()

	// indent only long option variation for alignement in help output
	s.SetConnectTimeout(cli.Option("    --connect-timeout").Duration("20ms"))
	s.SetPoolSize(cli.Option("-p, --pool-size").Uint16(200))
	c.srv = s
}

// Run listens for tcp connections. Blocks until context is cancelled
// or accepting a connection fails. Accepting new connection can only
// be interrupted if listener has SetDeadline method.
func (c *SrvCmd) Run(ctx context.Context) error {

	f := ttsrv.NewConnFeed()
	f.Bind = c.binds[0].String()
	f.AddConnection = c.srv.AddConnection
	f.SetDebug(c.debug)

	return f.Run(ctx)
}
