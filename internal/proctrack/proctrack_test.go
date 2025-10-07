//go:build darwin || linux
// +build darwin linux

package proctrack

import (
	"os/exec"
	"syscall"
	"testing"
	"time"
)

// TestRegisterGroup ensures that RegisterGroup returns a channel that is closed
// once every process in the registered group has exited.
func TestRegisterGroup(t *testing.T) {
	cmd := exec.Command("sleep", "0.2")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start command: %v", err)
	}

	doneCh := RegisterGroup(cmd.Process.Pid)

	select {
	case <-doneCh:
		// success
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for process group completion")
	}
}
