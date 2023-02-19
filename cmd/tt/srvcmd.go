package main

import (
	"github.com/gregoryv/cmdline"
	"github.com/gregoryv/tt/ttsrv"
)

type SrvCmd struct {
	debug bool

	*ttsrv.Server
}

func (c *SrvCmd) ExtraOptions(cli *cmdline.Parser) {
	var b ttsrv.BindConf
	b.URL = cli.Option("-b, --bind-tcp, $TT_BIND_TCP").Url("tcp://localhost:1883")
	b.AcceptTimeout = cli.Option("-a, --accept-timeout").Duration("1s")
	b.Debug = c.debug

	s := ttsrv.NewServer()
	s.AddBindConf(&b)

	// indent only long option variation for alignement in help output
	s.SetConnectTimeout(cli.Option("    --connect-timeout").Duration("20ms"))
	s.SetPoolSize(cli.Option("-p, --pool-size").Uint16(200))
	c.Server = s
}
