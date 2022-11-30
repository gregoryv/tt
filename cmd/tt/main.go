package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/gregoryv/cmdline"
)

func main() {
	var (
		cli = cmdline.NewBasicParser()
		// shared options

		// SubCmd commands
		commands = cli.Group("Commands", "COMMAND")

		_ = commands.New("pub", &PubCmd{})
		_ = commands.New("sub", &SubCmd{})
		_ = commands.New("srv", &SrvCmd{})

		cmd = commands.Selected()
	)
	u := cli.Usage()
	u.Preface(
		"mqtt-v5 server and client by Gregory Vinčić",
	)
	cli.Parse()

	sh := cmdline.DefaultShell
	if cmd == nil {
		// this shouldn't happen, default should be the first one. When testing it's empty
		log.Println("empty command", sh.Args())
		return
	}

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Kill, os.Interrupt)

	// closed when command execution returns
	done := make(chan struct{})

	// Run command in the background so we can interrupt it
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		defer close(done)
		if err := cmd.(Command).Run(ctx); err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}
			sh.Fatal(err)
		}
	}()

	// Handle interruptions gracefully
	select {
	case <-done:
		cancel()

	case <-interrupt:
		fmt.Println("interrupted")
		cancel()
		<-time.After(time.Millisecond)
		sh.Exit(0)
	}
}

type Command interface {
	Run(context.Context) error
}
