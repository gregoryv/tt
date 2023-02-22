package main

import (
	"bytes"
	"log"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/gregoryv/cmdline"
	"github.com/gregoryv/cmdline/clitest"
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
	host := "tcp://localhost:3881"
	log.Print(host)

	srv := exec.Command("tt", "srv", "-b", host)
	startCmd(t, srv)
	defer srv.Process.Kill()

	<-time.After(2 * time.Millisecond) // let it start

	{ // pub
		pub := exec.Command("tt", "pub", "-s", host)
		_ = pub.Run()
		if code := pub.ProcessState.ExitCode(); code != 0 {
			t.Fatalf("unexpected exit code %v", code)
		}
	}

	{ // pub 1
		pub := exec.Command("tt", "pub", "-q", "1", "-s", host)
		_ = pub.Run()
		if code := pub.ProcessState.ExitCode(); code != 0 {
			t.Fatalf("unexpected exit code %v", code)
		}
	}

	{ // pub 2
		pub := exec.Command("tt", "pub", "-q", "2", "-s", host)
		_ = pub.Run()
		if code := pub.ProcessState.ExitCode(); code != 0 {
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
	host := "tcp://localhost:3881"
	log.Print(host)

	srv := exec.Command("tt", "srv", "-b", host)
	startCmd(t, srv)
	defer srv.Process.Kill()

	<-time.After(2 * time.Millisecond) // let it start

	{ // sub
		sub := exec.Command("tt", "sub", "-s", host)
		var buf bytes.Buffer
		sub.Stdout = &buf
		sub.Start()

		<-time.After(2 * time.Millisecond) // let it start

		{ // publish something
			pub := exec.Command("tt", "pub", "-s", host)
			_ = pub.Run()
			if code := pub.ProcessState.ExitCode(); code != 0 {
				t.Fatalf("unexpected exit code %v", code)
			}
		}

		<-time.After(2 * time.Millisecond)
		if v := buf.String(); !strings.Contains(v, "PAYLOAD hug") {
			t.Error("missing logged payload", v)
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

func startCmd(t *testing.T, cmd *exec.Cmd) {
	t.Helper()
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
}
