// Command tt is a mqtt pub/sub client and broker
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
	if err := runCommand(cmd.(Command)); err != nil {
		log.Fatal(err)
	}
}

// shared
type opts struct {
	Debug        bool
	LogTimestamp bool
	ShowSettings bool
}

func runCommand(cmd Command) (err error) {
	intsig := make(chan os.Signal, 1)
	killsig := make(chan os.Signal, 1)
	signal.Notify(intsig, os.Interrupt)
	signal.Notify(killsig, os.Kill)

	// closed when command execution returns
	done := make(chan struct{})

	// Run command in the background so we can interrupt it
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		defer close(done)
		// note that the outer err is set here
		if err = cmd.Run(ctx); err != nil {
			if errors.Is(err, context.Canceled) {
				err = nil
			}
		}
	}()

	// Handle interruptions gracefully
	select {
	case <-done:
		cancel()

	case <-killsig:
		return fmt.Errorf("killed")

	case <-intsig:
		// this is ok
		cancel()
		<-time.After(time.Millisecond)
	}
	return
}

type Command interface {
	Run(context.Context) error
}
