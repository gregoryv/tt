package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/url"
	"strings"
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

	{ // pub
		sh := clitest.NewShellT("test", "pub", "-s", host)
		cmdline.DefaultShell = sh
		main()
		if code := sh.ExitCode; code != 0 {
			t.Fatalf("unexpected exit code %v", code)
		}
	}
}
func Test_main_sub(t *testing.T) {
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

	{ // sub
		sh := clitest.NewShellT("test", "sub", "-s", host)
		cmdline.DefaultShell = sh
		go main()
		<-time.After(2 * time.Millisecond) // let it start
		if code := sh.ExitCode; code != 0 {
			t.Fatalf("unexpected exit code %v", code)
		}
		var buf bytes.Buffer
		subWriter = &buf
		{ // publish something
			// let's use the pubcmd directly
			u, _ := url.Parse(host)
			c := &PubCmd{
				server:  u,
				topic:   "gopher/pink",
				payload: "hug",
				timeout: time.Second,
				//debug:    true,
				clientID: "test ",
			}
			c.Run(context.Background())
			<-time.After(2 * time.Millisecond)
			if v := buf.String(); !strings.Contains(v, "PAYLOAD hug") {
				t.Error("missing logged payload", v)
			}
		}
	}
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
