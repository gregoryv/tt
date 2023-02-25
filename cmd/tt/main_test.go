package main

import (
	"os/exec"
	"testing"
	"time"
)

func TestCommands(t *testing.T) {
	url := "tcp://localhost:3881"

	runCmd(t, exec.Command("tt", "-h"))

	startCmd(t, exec.Command("tt", "srv", "-b", url))
	startCmd(t, exec.Command("tt", "sub", "-s", url))

	runCmd(t, exec.Command("tt", "pub", "-s", url))
	runCmd(t, exec.Command("tt", "pub", "-q", "1", "-s", url))

	// following should fail
	notRun(t, exec.Command("tt", "badcmd"))
	notRun(t, exec.Command("tt", "pub", "-q", "2", "-s", url)) // should fail
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
