package main

import (
	"os/exec"
	"testing"
	"time"

	"github.com/gregoryv/cmdline"
	"github.com/gregoryv/cmdline/clitest"
)

func Test_main_help(t *testing.T) {
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

	startCmd(t, exec.Command("tt", "srv", "-b", url))
	startCmd(t, exec.Command("tt", "sub", "-s", url))

	runCmd(t, exec.Command("tt", "pub", "-s", url))
	runCmd(t, exec.Command("tt", "pub", "-q", "1", "-s", url))
	notRun(t, exec.Command("tt", "pub", "-q", "2", "-s", url)) // should fail
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

func notRun(t *testing.T, cmd *exec.Cmd) {
	t.Helper()
	if out, err := cmd.CombinedOutput(); err == nil {
		t.Error(cmd, "\n", string(out), err)
	}
}
