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
	"github.com/gregoryv/tt"
)

func main() {
	log.SetFlags(0)

	var (
		cli = cmdline.NewBasicParser()
		// shared options
		debug        = cli.Flag("-d, --debug")
		logTimestamp = cli.Flag("-T, --log-timestamp")
		showSettings = cli.Flag("-S, --show-settings")

		commands = cli.Group("Commands", "COMMAND")

		_ = commands.New("pub", &PubCmd{
			debug:        debug,
			showSettings: showSettings,
		})
		_ = commands.New("sub", &SubCmd{
			debug:        debug,
			output:       cmdline.DefaultShell.Stdout(),
			showSettings: showSettings,
		})
		_ = commands.New("srv", &SrvCmd{
			Server: &tt.Server{
				Debug:        debug,
				ShowSettings: showSettings,
			},
		})

		cmd = commands.Selected()
	)
	u := cli.Usage()
	u.Preface(
		"mqtt-v5 client and broker by Gregory Vinčić",
	)
	cli.Parse()

	if logTimestamp {
		log.SetFlags(log.Flags() | log.LstdFlags)
	}
	// run the selected command
	if err := runCommand(cmd); err != nil {
		// using DefaultShell.Fatal so we can verify the behavior
		cmdline.DefaultShell.Fatal(err)
	}
}

func runCommand(command any) (err error) {
	if command == nil {
		return fmt.Errorf("empty command")
	}

	cmd, ok := command.(Command)
	if !ok {
		return fmt.Errorf("%T is missing Run(context.Context) error", command)
	}

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
		if err = cmd.(Command).Run(ctx); err != nil {
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
