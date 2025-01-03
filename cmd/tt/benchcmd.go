package main

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"sync/atomic"
	"time"

	"github.com/gregoryv/cmdline"
	"github.com/gregoryv/mq"
	"github.com/gregoryv/tt"
	"github.com/gregoryv/tt/event"
)

type BenchCmd struct {
	shared opts

	server   *url.URL
	useCase  string
	clientID string
}

func (c *BenchCmd) ExtraOptions(cli *cmdline.Parser) {
	c.useCase = cli.Option("-u, --use-case").Enum(
		"singlenode-conn-tcp-10K-100", // default
		"singlenode-conn-tcp-10K-100",
	)
	c.server = cli.Option("-s, --server").Url("tcp://localhost:1883")
	c.clientID = cli.Option("-c, --client-id",
		"client id is used as prefix when multiple clients connect",
	).String("bench")
}

const clearLine = "\033[K"

func (c *BenchCmd) Run(ctx context.Context) error {

	switch c.useCase {
	case "singlenode-conn-tcp-10K-100":
		ctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
		failed := make(chan error)
		var connected atomic.Int64
		for i := 0; i < 10_000; i++ {
			select {
			case <-time.After(10 * time.Millisecond):
			case <-ctx.Done():
				return nil
			}

			go func(clientID string) {
				client := tt.NewClient()
				client.SetServer(c.server.String())
				client.SetDebug(c.shared.Debug)
				client.SetLogger(
					log.New(c.shared.LogOutput(), clientID+" ", log.Flags()),
				)
				go client.Run(ctx)

				for {
					select {
					case <-ctx.Done():
						p := mq.NewDisconnect()
						_ = client.Send(ctx, p)

					case v := <-client.Events():

						switch v := v.(type) {
						case event.ClientUp:
							p := mq.NewConnect()
							p.SetClientID(clientID)
							p.SetReceiveMax(1)
							p.SetKeepAlive(uint16(300))
							if err := client.Send(ctx, p); err != nil {
								failed <- err
								return
							}

						case *mq.ConnAck:
							switch v.ReasonCode() {
							case mq.Success:
								// ok
								connected.Add(1)
							default:
								failed <- fmt.Errorf(v.ReasonString())
								cancel()
								return
							}

						case event.ClientStop:
							if v.Err != nil {
								failed <- v.Err
							}
							return

						case event.ClientConnect, *mq.PingResp:
							// ignore

						default:
							failed <- fmt.Errorf("unexpected %T", v)
							return
						}
					}
				}
			}(fmt.Sprintf("%s-%05v", c.clientID, i))
		}
		fmt.Println("connected", connected.Load())
		// cleanup, ie. disconnect
		defer func() {
			cancel()
			<-time.After(time.Second)
		}()
		select {
		case <-ctx.Done():
			return nil

		case err := <-failed:
			fmt.Println(err)
			return err
		}
	}
	return fmt.Errorf("%q: no such use-case", c.useCase)
}
