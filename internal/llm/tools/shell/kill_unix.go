//go:build !windows

package shell

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

// killChildren terminates child processes on Unix systems using pgrep and SIGTERM
func (s *PersistentShell) killChildren() {
	if s.cmd == nil || s.cmd.Process == nil {
		return
	}

	// On Unix systems, use pgrep to find child processes
	pgrepCmd := exec.Command("pgrep", "-P", fmt.Sprintf("%d", s.cmd.Process.Pid))
	output, err := pgrepCmd.Output()
	if err != nil {
		return
	}

	// Parse pgrep output and send SIGTERM to each child process
	for _, pidStr := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if pidStr = strings.TrimSpace(pidStr); pidStr != "" {
			var pid int
			fmt.Sscanf(pidStr, "%d", &pid)
			if pid > 0 {
				proc, err := os.FindProcess(pid)
				if err == nil {
					proc.Signal(syscall.SIGTERM)
				}
			}
		}
	}
}
