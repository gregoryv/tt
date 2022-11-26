package main

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"testing"
	"time"

	"github.com/gregoryv/cmdline"
	"github.com/gregoryv/cmdline/clitest"
	"github.com/gregoryv/tt"
)

func Test_main_help(t *testing.T) {
	cmdline.DefaultShell = clitest.NewShellT("test", "-h")
	main()
}

func Test_main_pub(t *testing.T) {
	srv := tt.NewServer()
	go srv.Run(context.Background())

	<-time.After(2 * time.Millisecond) // let it start
	defer srv.Close()

	// then use
	u, err := url.Parse(fmt.Sprintf("%s://%s", srv.Addr().Network(), srv.Addr().String()))
	if err != nil {
		t.Fatal(err)
	}

	host := fmt.Sprintf("localhost:%v", u.Port())
	log.Print(host)

	cmdline.DefaultShell = clitest.NewShellT("test", "pub", "-s", host)
	main()
}

func Test_main_fails(t *testing.T) {
	// should fail
	sh := clitest.NewShellT("test", "pub", "-s", "nosuchthing:123")
	cmdline.DefaultShell = sh
	main()
	if sh.ExitCode != 1 {
		t.Fatal("pub should fail when bad server provided")
	}
}
