package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gregoryv/cmdline"
	"github.com/gregoryv/cmdline/clitest"
	"github.com/gregoryv/tt/ttsrv"
)

func Test_main_help(t *testing.T) {
	os.Args = []string{"test", "-h"}
	cmdline.DefaultShell = clitest.NewShellT("test", "-h")
	main()
}

func Test_main_srv(t *testing.T) {
	sh := clitest.NewShellT("test", "srv", "-b", "tcp://localhost:2883")
	cmdline.DefaultShell = sh
	sh.ExitCode = -1
	go main()
	<-time.After(10 * time.Millisecond)
	if v := sh.ExitCode; v != -1 {
		t.Error("srv command exited with", v)
	}
}

func Test_main_pub(t *testing.T) {
	srv := ttsrv.NewServer()
	ln := ttsrv.NewConnFeed()
	ln.SetServer(srv)
	go ln.Run(context.Background())

	<-time.After(2 * time.Millisecond) // let it start
	defer ln.Close()

	// then use
	u, err := url.Parse(fmt.Sprintf("%s://%s", ln.Addr().Network(), ln.Addr().String()))
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

	{ // pub 1
		sh := clitest.NewShellT("test", "pub", "-q", "1", "-s", host)
		cmdline.DefaultShell = sh
		main()
		if code := sh.ExitCode; code != 0 {
			t.Fatalf("unexpected exit code %v", code)
		}
	}

	{ // pub 2
		sh := clitest.NewShellT("test", "pub", "-q", "2", "-s", host)
		cmdline.DefaultShell = sh
		main()
		if code := sh.ExitCode; code != 0 {
			t.Fatalf("unexpected exit code %v", code)
		}
	}
}

func Test_subFailsOnBadHost(t *testing.T) {
	sh := clitest.NewShellT("test", "sub", "-s", "__")
	cmdline.DefaultShell = sh
	main()
	if sh.ExitCode == 0 {
		t.Error("expected sub to fail on bad host")
	}
}

func Test_main_sub(t *testing.T) {
	srv := ttsrv.NewServer()
	ln := ttsrv.NewConnFeed()
	ln.SetServer(srv)
	go ln.Run(context.Background())

	<-time.After(2 * time.Millisecond) // let it start
	defer ln.Close()

	// then use
	u, err := url.Parse(fmt.Sprintf("%s://%s", ln.Addr().Network(), ln.Addr().String()))
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

// disabled once we added feature to interrupt commands gracefully
func Test_main_fails(t *testing.T) {
	// should fail
	sh := clitest.NewShellT("test", "pub", "-s", "nosuchthing:123")
	cmdline.DefaultShell = sh
	main()
	if sh.ExitCode != 1 {
		t.Fatal("pub should fail when bad server provided")
	}
}
