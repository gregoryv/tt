package main

import (
	"context"

	"github.com/gregoryv/cmdline"
	"github.com/gregoryv/tt"
)

type SrvCmd struct {
	*tt.Listener
}

func (c *SrvCmd) ExtraOptions(cli *cmdline.Parser) {
	s := tt.NewServer()
	s.SetConnectTimeout(cli.Option("-c, --connect-timeout").Duration("20ms"))
	s.PoolSize = cli.Option("-p, --pool-size").Uint16(200)

	b := tt.NewListener()
	b.Bind = cli.Option("-b, --bind-tcp, $TT_BIND_TCP").String("tcp://localhost:1883")
	b.AcceptTimeout = cli.Option("-a, --accept-timeout").Duration("1ms")
	b.AddConnection = s.AddConnection

	c.Listener = b
}

// Run listens for tcp connections. Blocks until context is cancelled
// or accepting a connection fails. Accepting new connection can only
// be interrupted if listener has SetDeadline method.
func (c *SrvCmd) Run(ctx context.Context) error {
	return c.Listener.Run(ctx)
}
