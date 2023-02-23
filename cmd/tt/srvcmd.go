package main

import (
	"log"
	"os"

	"github.com/gregoryv/cmdline"
	"github.com/gregoryv/tt"
)

type SrvCmd struct {
	debug bool

	*tt.Server
}

func (c *SrvCmd) ExtraOptions(cli *cmdline.Parser) {
	c.Server.Debug = c.debug
	c.Server.Logger = log.New(os.Stderr, "ttsrv ", log.Flags())

	b := c.Binds[0] // assuming there is one bind at least
	b.URL = cli.Option("-b, --bind-tcp, $TT_BIND_TCP").Url(b.URL.String())
	b.AcceptTimeout = cli.Option("-a, --accept-timeout").Duration(b.AcceptTimeout.String())

	// Server settings
	// indent only long option variation for alignement in help output
	c.ConnectTimeout = cli.Option("    --connect-timeout").Duration("200ms")
}
