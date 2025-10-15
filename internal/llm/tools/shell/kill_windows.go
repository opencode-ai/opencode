//go:build windows

package shell

import (
	"fmt"
	"os/exec"
	"syscall"
	"time"
)

var (
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	generateConsoleCtrlEvent = kernel32.NewProc("GenerateConsoleCtrlEvent")
)

const (
	CTRL_BREAK_EVENT = 1
)

// killChildren terminates child processes on Windows systems
// Uses tasklist + taskkill for comprehensive termination, with fallback to Process.Kill
// and console control events when the shell is still alive
func (s *PersistentShell) killChildren() {
	if s.cmd == nil || s.cmd.Process == nil {
		return
	}

	// First attempt: Use taskkill to terminate the process tree
	// This is more reliable for killing all child processes recursively
	taskkillCmd := exec.Command("taskkill", "/F", "/T", "/PID", fmt.Sprintf("%d", s.cmd.Process.Pid))
	err := taskkillCmd.Run()
	
	// If taskkill fails or if we need more control, try alternative approaches
	if err != nil || s.isAlive {
		// If the shell is still alive, try to send console control event first
		// This allows for graceful termination of console applications
		if s.isAlive {
			// Send CTRL_BREAK_EVENT to the process group
			// This propagates to children and allows them to handle the signal gracefully
			generateConsoleCtrlEvent.Call(uintptr(CTRL_BREAK_EVENT), uintptr(0))
			
			// Give processes a moment to handle the break event gracefully
			time.Sleep(100 * time.Millisecond)
		}
		
		// Fallback: Use Process.Kill for direct termination
		if s.cmd.Process != nil {
			s.cmd.Process.Kill()
		}
	}
}
