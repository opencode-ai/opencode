//go:build windows

package shell

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestStdoutCaptureWindows tests stdout capture for various Windows shells
func TestStdoutCaptureWindows(t *testing.T) {
	tests := []struct {
		name        string
		command     string
		expectedOut string
	}{
		{"CMD echo", "echo Hello CMD", "Hello CMD"},
		{"PowerShell Write-Output", "Write-Output 'Hello PowerShell'", "Hello PowerShell"},
		{"CMD dir command", "dir /B C:\\Windows\\System32\\kernel32.dll", "kernel32.dll"},
		{"Multi-line output", "echo Line1 && echo Line2", "Line1"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			shell := GetPersistentShell("C:\\Windows\\System32")
			defer shell.Close()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			stdout, stderr, exitCode, interrupted, err := shell.Exec(ctx, test.command, 10000)

			if err != nil {
				t.Fatalf("Exec failed: %v", err)
			}

			if interrupted {
				t.Errorf("Unexpected interruption")
			}

			if exitCode != 0 {
				t.Errorf("Unexpected exit code: %d, stderr: %s", exitCode, stderr)
			}

			if !strings.Contains(stdout, test.expectedOut) {
				t.Errorf("Expected stdout to contain %q, got: %q", test.expectedOut, stdout)
			}
		})
	}
}

// TestStderrCaptureWindows tests stderr capture for various error conditions
func TestStderrCaptureWindows(t *testing.T) {
	tests := []struct {
		name         string
		command      string
		expectedCode int
	}{
		{"Non-existent command", "nonexistentcommand123", 1},
		{"Invalid directory", "dir C:\\NonExistentDirectory123", 1},
		{"PowerShell error", "Get-Process -Name NonExistentProcess123", 1},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			shell := GetPersistentShell("C:\\Windows\\System32")
			defer shell.Close()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			stdout, stderr, exitCode, interrupted, err := shell.Exec(ctx, test.command, 10000)

			// Command execution might succeed but return non-zero exit code
			if err != nil && !interrupted {
				// Some errors are expected for invalid commands
				t.Logf("Command failed as expected: %v", err)
			}

			if interrupted {
				t.Errorf("Unexpected interruption")
			}

			if exitCode == 0 {
				t.Errorf("Expected non-zero exit code, got: %d, stdout: %s, stderr: %s", exitCode, stdout, stderr)
			}

			// For error cases, we should get either stderr output or error in stdout
			if stderr == "" && !strings.Contains(strings.ToLower(stdout), "error") && !strings.Contains(strings.ToLower(stdout), "not found") {
				t.Logf("Warning: No error output captured for command: %s, stdout: %s, stderr: %s", test.command, stdout, stderr)
			}
		})
	}
}

// TestTimeoutWindows tests command timeout behavior
func TestTimeoutWindows(t *testing.T) {
	tests := []struct {
		name    string
		command string
		timeout int
	}{
		{"CMD timeout", "timeout /t 3 /nobreak", 1000},
		{"PowerShell Start-Sleep", "Start-Sleep -Seconds 3", 1000},
		{"Ping timeout", "ping -n 5 127.0.0.1", 1000},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			shell := GetPersistentShell("C:\\Windows\\System32")
			defer shell.Close()

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			start := time.Now()
			_, _, exitCode, interrupted, _ := shell.Exec(ctx, test.command, test.timeout)
			duration := time.Since(start)

			if !interrupted {
				t.Errorf("Expected interruption due to timeout")
			}

			// Should timeout around the specified timeout duration
			expectedDuration := time.Duration(test.timeout) * time.Millisecond
			if duration < expectedDuration || duration > expectedDuration+2*time.Second {
				t.Errorf("Expected duration around %v, got %v", expectedDuration, duration)
			}

			// Exit code should indicate interruption
			if exitCode != 143 && exitCode != 1 {
				t.Logf("Timeout exit code: %d", exitCode)
			}
		})
	}
}

// TestInterruptionWindows tests context cancellation behavior
func TestInterruptionWindows(t *testing.T) {
	shell := GetPersistentShell("C:\\Windows\\System32")
	defer shell.Close()

	ctx, cancel := context.WithCancel(context.Background())

	// Start a long-running command
	go func() {
		time.Sleep(500 * time.Millisecond)
		cancel() // Cancel the context
	}()

	start := time.Now()
	_, _, _, interrupted, _ := shell.Exec(ctx, "timeout /t 10 /nobreak", 0) // No timeout, rely on context
	duration := time.Since(start)

	if !interrupted {
		t.Errorf("Expected interruption due to context cancellation")
	}

	// Should be interrupted quickly after context cancellation
	if duration > 2*time.Second {
		t.Errorf("Expected quick interruption, took %v", duration)
	}
}

// TestCwdUpdatesWindows tests working directory changes
func TestCwdUpdatesWindows(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "shell_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	shell := GetPersistentShell("C:\\Windows\\System32")
	defer shell.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Change to temp directory
	cdCommand := fmt.Sprintf("cd /d %s", tempDir)
	stdout, stderr, exitCode, interrupted, err := shell.Exec(ctx, cdCommand, 5000)

	if err != nil {
		t.Fatalf("CD command failed: %v", err)
	}

	if interrupted {
		t.Errorf("Unexpected interruption")
	}

	if exitCode != 0 {
		t.Errorf("CD command failed with exit code: %d, stdout: %s, stderr: %s", exitCode, stdout, stderr)
	}

	// Verify we're in the new directory
	stdout, stderr, exitCode, interrupted, err = shell.Exec(ctx, "cd", 5000)

	if err != nil {
		t.Fatalf("PWD command failed: %v", err)
	}

	if interrupted {
		t.Errorf("Unexpected interruption")
	}

	if exitCode != 0 {
		t.Errorf("PWD command failed with exit code: %d, stderr: %s", exitCode, stderr)
	}

	// Check if the output contains our temp directory path
	if !strings.Contains(stdout, tempDir) {
		t.Errorf("Expected current directory to be %s, got: %s", tempDir, stdout)
	}

	// Test creating a file in the current directory
	testFile := "test_file.txt"
	stdout, stderr, exitCode, interrupted, err = shell.Exec(ctx, fmt.Sprintf("echo test content > %s", testFile), 5000)

	if err != nil {
		t.Fatalf("File creation failed: %v", err)
	}

	if interrupted {
		t.Errorf("Unexpected interruption")
	}

	if exitCode != 0 {
		t.Errorf("File creation failed with exit code: %d, stderr: %s", exitCode, stderr)
	}

	// Verify the file was created in the correct directory
	expectedPath := filepath.Join(tempDir, testFile)
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("Expected file %s to be created", expectedPath)
	}
}

// TestKillChildrenWindows tests the killChildren functionality
func TestKillChildrenWindows(t *testing.T) {
	shell := GetPersistentShell("C:\\Windows\\System32")
	defer shell.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Start a command that creates child processes
	// Using a batch command that starts multiple processes
	cmd := "timeout /t 10 /nobreak & timeout /t 10 /nobreak & timeout /t 10 /nobreak"

	start := time.Now()
	_, _, _, interrupted, _ := shell.Exec(ctx, cmd, 0)
	duration := time.Since(start)

	if !interrupted {
		t.Errorf("Expected interruption due to timeout")
	}

	// Should be killed relatively quickly by killChildren
	if duration > 3*time.Second {
		t.Errorf("killChildren took too long: %v", duration)
	}

	// Test that the shell is still alive after killing children
	if !shell.isAlive {
		t.Errorf("Shell should still be alive after killing children")
	}

	// Test that we can still execute commands
	ctx2, cancel2 := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel2()

	stdout, stderr, exitCode, interrupted, err := shell.Exec(ctx2, "echo Shell still works", 5000)

	if err != nil {
		t.Fatalf("Command after killChildren failed: %v", err)
	}

	if interrupted {
		t.Errorf("Unexpected interruption")
	}

	if exitCode != 0 {
		t.Errorf("Command after killChildren failed with exit code: %d, stderr: %s", exitCode, stderr)
	}

	if !strings.Contains(stdout, "Shell still works") {
		t.Errorf("Expected stdout to contain 'Shell still works', got: %s", stdout)
	}
}

// TestPowerShellSpecificWindows tests PowerShell-specific features
func TestPowerShellSpecificWindows(t *testing.T) {
	// This test specifically targets PowerShell functionality
	shell := GetPersistentShell("C:\\Windows\\System32")
	defer shell.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tests := []struct {
		name    string
		command string
		check   func(stdout, stderr string, exitCode int) bool
	}{
		{
			"PowerShell Get-Process",
			"Get-Process | Select-Object -First 1 | Format-Table Name",
			func(stdout, stderr string, exitCode int) bool {
				return exitCode == 0 && (strings.Contains(stdout, "Name") || strings.Contains(stdout, "ProcessName"))
			},
		},
		{
			"PowerShell variables",
			"$test = 'Hello PowerShell'; Write-Output $test",
			func(stdout, stderr string, exitCode int) bool {
				return exitCode == 0 && strings.Contains(stdout, "Hello PowerShell")
			},
		},
		{
			"PowerShell error handling",
			"try { Get-Item 'C:\\NonExistentFile123.txt' } catch { Write-Output 'Caught error' }",
			func(stdout, stderr string, exitCode int) bool {
				return strings.Contains(stdout, "Caught error") || strings.Contains(stderr, "cannot find")
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			stdout, stderr, exitCode, interrupted, err := shell.Exec(ctx, test.command, 10000)

			if err != nil {
				t.Fatalf("PowerShell command failed: %v", err)
			}

			if interrupted {
				t.Errorf("Unexpected interruption")
			}

			if !test.check(stdout, stderr, exitCode) {
				t.Errorf("PowerShell test failed: %s\nStdout: %s\nStderr: %s\nExit code: %d",
					test.name, stdout, stderr, exitCode)
			}
		})
	}
}

// TestCmdSpecificWindows tests CMD-specific features
func TestCmdSpecificWindows(t *testing.T) {
	// This test specifically targets CMD functionality
	shell := GetPersistentShell("C:\\Windows\\System32")
	defer shell.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tests := []struct {
		name    string
		command string
		check   func(stdout, stderr string, exitCode int) bool
	}{
		{
			"CMD environment variables",
			"set TEST_VAR=Hello CMD && echo %TEST_VAR%",
			func(stdout, stderr string, exitCode int) bool {
				return exitCode == 0 && strings.Contains(stdout, "Hello CMD")
			},
		},
		{
			"CMD pipes",
			"echo Hello World | findstr Hello",
			func(stdout, stderr string, exitCode int) bool {
				return exitCode == 0 && strings.Contains(stdout, "Hello World")
			},
		},
		{
			"CMD error code",
			"exit /b 42",
			func(stdout, stderr string, exitCode int) bool {
				return exitCode == 42
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			stdout, stderr, exitCode, interrupted, err := shell.Exec(ctx, test.command, 10000)

			if err != nil && exitCode == 0 {
				t.Fatalf("CMD command failed: %v", err)
			}

			if interrupted {
				t.Errorf("Unexpected interruption")
			}

			if !test.check(stdout, stderr, exitCode) {
				t.Errorf("CMD test failed: %s\nStdout: %s\nStderr: %s\nExit code: %d",
					test.name, stdout, stderr, exitCode)
			}
		})
	}
}
