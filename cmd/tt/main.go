// Command tt is a mqtt pub/sub client and broker
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/gregoryv/cmdline"
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
}

type Runner interface {
	Run(context.Context) error
}
