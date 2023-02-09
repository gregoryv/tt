package ttsrv_test

import (
	"context"
	"log"
	"net"

	"github.com/gregoryv/tt/ttsrv"
)

// Example shows how to run the provided server.
func Example_server() {
	srv := ttsrv.NewServer()
	ln := ttsrv.NewListener()
	ln.SetServer(srv)
	go ln.Run(context.Background())

	<-ln.Up
	// then use
	a := ln.Addr()
	conn, err := net.Dial(a.Network(), a.String())
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	_ = conn // use it in your client
}
