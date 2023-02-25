package main

import (
	"log"
	"os"

	"github.com/gregoryv/cmdline"
	"github.com/gregoryv/tt"
)

type SrvCmd struct {
	*tt.Server
}

func (c *SrvCmd) ExtraOptions(cli *cmdline.Parser) {
	c.Server.Logger = log.New(os.Stderr, "ttsrv ", log.Flags())

	var b tt.Bind
	{
		v := cli.Option("-b, --bind-tcp, $TT_BIND_TCP").Url("tcp://localhost:")
		b.URL = v.String()
	}
	{
		v := cli.Option("-a, --accept-timeout").Duration("500ms")
		b.AcceptTimeout = v.String()
	}
	c.Server.Binds = append(c.Server.Binds, &b)

	// Server settings
	// indent only long option variation for alignement in help output
	c.ConnectTimeout = cli.Option("    --connect-timeout").Duration("200ms")
}
