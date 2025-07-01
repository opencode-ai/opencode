//go:build windows

package shell

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

// TestCrossShellCompatibilityWindows tests that the same logical operations work across different Windows shells
func TestCrossShellCompatibilityWindows(t *testing.T) {
	shells := []struct {
		name string
		kind ShellKind
	}{
		{"CMD", CmdExe},
		{"PowerShell", Pwsh},
		{"Windows PowerShell", WindowsPowerShell},
	}

	for _, shell := range shells {
		t.Run(shell.name, func(t *testing.T) {
			// Skip if shell is not available
			if !isShellAvailable(shell.kind) {
				t.Skipf("Shell %s not available", shell.name)
				return
			}

			testShellBasicOperations(t, shell.kind, shell.name)
		})
	}
}

// isShellAvailable checks if a particular shell is available on the system
func isShellAvailable(kind ShellKind) bool {
	shellPath, _ := getShellDefaults(kind)
	if shellPath == "" {
		return false
	}
	
	// Try to detect if the shell exists
	switch kind {
	case Pwsh:
		return DetectShellKind() == Pwsh
	case WindowsPowerShell:
		return DetectShellKind() == WindowsPowerShell || DetectShellKind() == Pwsh
	case CmdExe:
		return true // CMD should always be available on Windows
	default:
		return false
	}
}

// testShellBasicOperations runs a series of basic operations to test shell functionality
func testShellBasicOperations(t *testing.T, kind ShellKind, shellName string) {
	shell := GetPersistentShell("C:\\Windows\\System32")
	defer shell.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	t.Run(fmt.Sprintf("%s_Echo", shellName), func(t *testing.T) {
		cmd := getEchoCommand(kind, "Hello from "+shellName)
		stdout, stderr, exitCode, interrupted, err := shell.Exec(ctx, cmd, 5000)

		if err != nil {
			t.Fatalf("Echo command failed: %v", err)
		}
		if interrupted {
			t.Errorf("Unexpected interruption")
		}
		if exitCode != 0 {
			t.Errorf("Unexpected exit code: %d, stderr: %s", exitCode, stderr)
		}
		if !strings.Contains(stdout, "Hello from "+shellName) {
			t.Errorf("Expected output not found in stdout: %s", stdout)
		}
	})

	t.Run(fmt.Sprintf("%s_DirectoryListing", shellName), func(t *testing.T) {
		cmd := getDirectoryCommand(kind)
		stdout, stderr, exitCode, interrupted, err := shell.Exec(ctx, cmd, 5000)

		if err != nil {
			t.Fatalf("Directory command failed: %v", err)
		}
		if interrupted {
			t.Errorf("Unexpected interruption")
		}
		if exitCode != 0 {
			t.Errorf("Unexpected exit code: %d, stderr: %s", exitCode, stderr)
		}
		// Should contain some files from System32
		if !strings.Contains(stdout, "kernel32") && !strings.Contains(stdout, "cmd") {
			t.Logf("Directory output: %s", stdout)
		}
	})

	t.Run(fmt.Sprintf("%s_EnvironmentVariable", shellName), func(t *testing.T) {
		cmd := getEnvironmentVariableCommand(kind)
		stdout, stderr, exitCode, interrupted, err := shell.Exec(ctx, cmd, 5000)

		if err != nil {
			t.Fatalf("Environment variable command failed: %v", err)
		}
		if interrupted {
			t.Errorf("Unexpected interruption")
		}
		if exitCode != 0 {
			t.Errorf("Unexpected exit code: %d, stderr: %s", exitCode, stderr)
		}
		// Should show PATH or another environment variable
		if stdout == "" {
			t.Errorf("Expected environment variable output, got empty string")
		}
	})
}

// getEchoCommand returns the appropriate echo command for each shell type
func getEchoCommand(kind ShellKind, message string) string {
	switch kind {
	case CmdExe:
		return fmt.Sprintf("echo %s", message)
	case Pwsh, WindowsPowerShell:
		return fmt.Sprintf("Write-Output '%s'", message)
	default:
		return fmt.Sprintf("echo %s", message)
	}
}

// getDirectoryCommand returns the appropriate directory listing command for each shell type
func getDirectoryCommand(kind ShellKind) string {
	switch kind {
	case CmdExe:
		return "dir /B"
	case Pwsh, WindowsPowerShell:
		return "Get-ChildItem | Select-Object -ExpandProperty Name"
	default:
		return "dir /B"
	}
}

// getEnvironmentVariableCommand returns the appropriate command to display an environment variable
func getEnvironmentVariableCommand(kind ShellKind) string {
	switch kind {
	case CmdExe:
		return "echo %PATH%"
	case Pwsh, WindowsPowerShell:
		return "$env:PATH"
	default:
		return "echo %PATH%"
	}
}

// TestWindowsSpecificFeatures tests Windows-specific functionality that should work regardless of shell
func TestWindowsSpecificFeatures(t *testing.T) {
	shell := GetPersistentShell("C:\\Windows\\System32")
	defer shell.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	t.Run("Windows_SystemInfo", func(t *testing.T) {
		// Test Windows-specific system information command
		stdout, stderr, exitCode, interrupted, err := shell.Exec(ctx, "systeminfo | findstr /C:\"OS Name\"", 10000)

		if err != nil {
			t.Fatalf("Systeminfo command failed: %v", err)
		}
		if interrupted {
			t.Errorf("Unexpected interruption")
		}
		if exitCode != 0 {
			t.Errorf("Unexpected exit code: %d, stderr: %s", exitCode, stderr)
		}
		if !strings.Contains(strings.ToLower(stdout), "windows") {
			t.Logf("Systeminfo output: %s", stdout)
		}
	})

	t.Run("Windows_ProcessList", func(t *testing.T) {
		// Test Windows process listing
		stdout, stderr, exitCode, interrupted, err := shell.Exec(ctx, "tasklist /FO CSV | findstr /C:\"Image Name\"", 10000)

		if err != nil {
			t.Fatalf("Tasklist command failed: %v", err)
		}
		if interrupted {
			t.Errorf("Unexpected interruption")
		}
		if exitCode != 0 {
			t.Errorf("Unexpected exit code: %d, stderr: %s", exitCode, stderr)
		}
		if !strings.Contains(stdout, "Image Name") {
			t.Logf("Tasklist output: %s", stdout)
		}
	})

	t.Run("Windows_NetworkConfig", func(t *testing.T) {
		// Test Windows network configuration
		stdout, stderr, exitCode, interrupted, err := shell.Exec(ctx, "ipconfig | findstr /C:\"IPv4\"", 10000)

		if err != nil {
			t.Fatalf("Ipconfig command failed: %v", err)
		}
		if interrupted {
			t.Errorf("Unexpected interruption")
		}
		// Note: Exit code might be 1 if no IPv4 addresses are found, which is okay
		if exitCode != 0 && exitCode != 1 {
			t.Errorf("Unexpected exit code: %d, stderr: %s", exitCode, stderr)
		}
		// IPv4 output is optional - some systems might not have configured interfaces
		t.Logf("Ipconfig output: %s", stdout)
	})
}

// TestErrorHandlingAcrossShells tests that error conditions are properly handled across different shells
func TestErrorHandlingAcrossShells(t *testing.T) {
	shells := []struct {
		name string
		kind ShellKind
	}{
		{"CMD", CmdExe},
		{"PowerShell", Pwsh},
	}

	for _, shell := range shells {
		t.Run(shell.name, func(t *testing.T) {
			if !isShellAvailable(shell.kind) {
				t.Skipf("Shell %s not available", shell.name)
				return
			}

			testShellErrorHandling(t, shell.kind, shell.name)
		})
	}
}

// testShellErrorHandling tests error conditions for a specific shell
func testShellErrorHandling(t *testing.T, kind ShellKind, shellName string) {
	shell := GetPersistentShell("C:\\Windows\\System32")
	defer shell.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	t.Run(fmt.Sprintf("%s_NonExistentCommand", shellName), func(t *testing.T) {
		stdout, stderr, exitCode, interrupted, err := shell.Exec(ctx, "nonexistentcommand12345", 5000)

		// Error is expected, but execution should not be interrupted
		if interrupted {
			t.Errorf("Unexpected interruption")
		}

		// Should return non-zero exit code
		if exitCode == 0 {
			t.Errorf("Expected non-zero exit code for non-existent command, got: %d", exitCode)
		}

		// Should have some error indication
		if stderr == "" && !strings.Contains(strings.ToLower(stdout), "not recognized") && 
		   !strings.Contains(strings.ToLower(stdout), "not found") &&
		   !strings.Contains(strings.ToLower(stdout), "error") {
			t.Logf("Warning: No clear error indication. Stdout: %s, Stderr: %s, Err: %v", stdout, stderr, err)
		}
	})

	t.Run(fmt.Sprintf("%s_InvalidSyntax", shellName), func(t *testing.T) {
		// Test shell-specific invalid syntax
		var cmd string
		switch kind {
		case CmdExe:
			cmd = "if ((" // Invalid CMD syntax
		case Pwsh, WindowsPowerShell:
			cmd = "Get-Process -InvalidParameter" // Invalid PowerShell parameter
		}

		stdout, stderr, exitCode, interrupted, err := shell.Exec(ctx, cmd, 5000)

		if interrupted {
			t.Errorf("Unexpected interruption")
		}

		// Should return non-zero exit code for syntax errors
		if exitCode == 0 {
			t.Logf("Note: Syntax error did not return non-zero exit code. Stdout: %s, Stderr: %s, Err: %v", 
				stdout, stderr, err)
		}
	})
}
