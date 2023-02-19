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
	b := c.Binds[0] // assuming there is one bind at least
	b.URL = cli.Option("-b, --bind-tcp, $TT_BIND_TCP").Url(b.URL.String())
	b.AcceptTimeout = cli.Option("-a, --accept-timeout").Duration(b.AcceptTimeout.String())
	b.Debug = c.debug

	// Server settings
	// indent only long option variation for alignement in help output
	c.ConnectTimeout = cli.Option("    --connect-timeout").Duration("200ms")
	c.PoolSize = cli.Option("-p, --pool-size").Uint16(200)
}
