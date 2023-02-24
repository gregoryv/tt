package main

import (
	"os"
	"os/exec"
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

func TestCommands(t *testing.T) {
	url := "tcp://localhost:3881"

	srv := exec.Command("tt", "srv", "-b", url)
	startCmd(t, srv)

	sub := exec.Command("tt", "sub", "-s", url)
	startCmd(t, sub)

	pub := exec.Command("tt", "pub", "-s", url)
	runCmd(t, pub)
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
	t.Cleanup(func() {
		cmd.Process.Kill()
		cmd.Process.Wait()
	})
	<-time.After(2 * time.Millisecond) // let it start
}

func runCmd(t *testing.T, cmd *exec.Cmd) {
	t.Helper()
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatal(string(out), err)
	}
	if v := cmd.ProcessState.ExitCode(); v != 0 {
		t.Fatalf("unexpected exit code %v", v)
	}
}
