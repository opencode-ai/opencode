//go:build windows

package shell

import (
	"context"
	"strings"
	"testing"
	"time"
)

// TestSimpleWindowsExecution demonstrates basic Windows command execution
func TestSimpleWindowsExecution(t *testing.T) {
	t.Log("Testing basic Windows command execution...")
	
	shell := GetPersistentShell("C:\\Windows\\System32")
	if shell == nil {
		t.Fatal("Failed to create persistent shell")
	}
	defer shell.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test basic echo command
	stdout, stderr, exitCode, interrupted, err := shell.Exec(ctx, "echo Hello Windows", 5000)
	
	t.Logf("Command: echo Hello Windows")
	t.Logf("Stdout: %q", stdout)
	t.Logf("Stderr: %q", stderr)
	t.Logf("Exit code: %d", exitCode)
	t.Logf("Interrupted: %v", interrupted)
	t.Logf("Error: %v", err)
	
	if err != nil {
		t.Errorf("Basic echo command failed: %v", err)
	}
	
	if !strings.Contains(stdout, "Hello Windows") {
		t.Errorf("Expected 'Hello Windows' in stdout, got: %q", stdout)
	}
}

// TestWindowsShellDetection demonstrates shell detection on Windows
func TestWindowsShellDetection(t *testing.T) {
	t.Log("Testing Windows shell detection...")
	
	kind := DetectShellKind()
	t.Logf("Detected shell kind: %s", kind.String())
	
	// On Windows, should detect one of the Windows shells
	switch kind {
	case Pwsh, WindowsPowerShell, CmdExe:
		t.Logf("Successfully detected Windows shell: %s", kind.String())
	default:
		t.Errorf("Expected Windows shell, got: %s", kind.String())
	}
}
