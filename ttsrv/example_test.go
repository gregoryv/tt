package ttsrv_test

import (
	"context"
	"fmt"

	"github.com/gregoryv/tt/ttsrv"
)

// Example shows how to run the provided server.
func Example_server() {
	s := ttsrv.NewServer()
	go s.Run(context.Background())

	b := s.Binds[0] // by default there is one
	fmt.Println(b.URL)

	// output:
	// tcp://localhost:
}
