package main

import (
	"context"
	"net"

	"github.com/gregoryv/cmdline"
	"github.com/gregoryv/tt"
)

type ServeCmd struct {
	*tt.Server
}

func (c *ServeCmd) ExtraOptions(cli *cmdline.Parser) {
	s := tt.NewServer()
	s.Bind = cli.Option("-b, --bind, $BIND").String("localhost:1883")
	s.AcceptTimeout = cli.Option("-a, --accept-timeout").Duration("1ms")
	s.ConnectTimeout = cli.Option("-c, --connect-timeout").Duration("20ms")
	s.PoolSize = cli.Option("-p, --pool-size").Uint16(200)

	c.Server = s
}

// Run listens for tcp connections. Blocks until context is cancelled
// or accepting a connection fails. Accepting new connection can only
// be interrupted if listener has SetDeadline method.
func (c *ServeCmd) Run(ctx context.Context) error {
	ln, err := net.Listen("tcp", c.Bind)
	if err != nil {
		return err
	}
	c.Server.Listener = ln
	return c.Server.Run(ctx)
}
