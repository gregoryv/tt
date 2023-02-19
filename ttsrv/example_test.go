package ttsrv_test

import (
	"context"
	"log"
	"net"
	"time"

	"github.com/gregoryv/tt/ttsrv"
)

// Example shows how to run the provided server.
func Example_server() {
	s := ttsrv.NewServer()
	b, _ := ttsrv.NewBindConf("tcp://:", "20s")
	s.Binds = append(s.Binds, b)

	go s.Run(context.Background())

	<-time.After(time.Millisecond)
	conn, err := net.Dial(b.URL.Scheme, b.URL.Host)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	// use it in your client
}
