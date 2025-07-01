//go:build !windows

package shell

import (
	"context"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestUnixCriticalFunctionality(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix-specific test")
	}

	tempDir, err := os.MkdirTemp("", "opencode-shell-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	shell := newPersistentShell(tempDir)
	if shell == nil {
		t.Fatal("Failed to create persistent shell")
	}
	defer shell.Close()

	ctx := context.Background()

	// Test 1: Basic shell functionality
	t.Run("Basic Commands", func(t *testing.T) {
		stdout, stderr, exitCode, interrupted, err := shell.Exec(ctx, "echo 'Hello Unix'", 5000)
		if err != nil {
			t.Fatalf("Failed to execute echo: %v", err)
		}
		if exitCode != 0 {
			t.Errorf("echo failed with exit code %d, stderr: %s", exitCode, stderr)
		}
		if interrupted {
			t.Error("echo command was interrupted")
		}
		if !strings.Contains(stdout, "Hello Unix") {
			t.Errorf("Expected 'Hello Unix' in output, got: %q", stdout)
		}
	})

	// Test 2: Shell persistence (directory changes)
	t.Run("Directory Persistence", func(t *testing.T) {
		// Create a subdirectory
		subDir := tempDir + "/testdir"
		os.Mkdir(subDir, 0755)

		// Change to subdirectory
		stdout, stderr, exitCode, _, err := shell.Exec(ctx, "cd testdir", 5000)
		if err != nil {
			t.Fatalf("Failed to execute cd: %v", err)
		}
		if exitCode != 0 {
			t.Errorf("cd failed with exit code %d, stderr: %s", exitCode, stderr)
		}

		// Verify we're in subdirectory
		stdout, stderr, exitCode, _, err = shell.Exec(ctx, "pwd", 5000)
		if err != nil {
			t.Fatalf("Failed to execute pwd: %v", err)
		}
		if exitCode != 0 {
			t.Errorf("pwd failed with exit code %d, stderr: %s", exitCode, stderr)
		}
		if !strings.Contains(stdout, "testdir") {
			t.Errorf("Should be in testdir, got: %q", stdout)
		}
	})

	// Test 3: Environment variable persistence
	t.Run("Environment Persistence", func(t *testing.T) {
		// Set environment variable
		stdout, stderr, exitCode, _, err := shell.Exec(ctx, "export TESTVAR=testvalue", 5000)
		if err != nil {
			t.Fatalf("Failed to execute export: %v", err)
		}
		
		// Check environment variable
		stdout, stderr, exitCode, _, err = shell.Exec(ctx, "echo $TESTVAR", 5000)
		if err != nil {
			t.Fatalf("Failed to execute echo: %v", err)
		}
		if exitCode != 0 {
			t.Errorf("echo failed with exit code %d, stderr: %s", exitCode, stderr)
		}
		if !strings.Contains(stdout, "testvalue") {
			t.Errorf("Environment variable not persisted, got: %q", stdout)
		}
	})

	// Test 4: Signal handling with timeout
	t.Run("Signal Handling", func(t *testing.T) {
		// Create a context with a short timeout
		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		// Execute a command that would run longer than the timeout
		stdout, stderr, exitCode, interrupted, err := shell.Exec(ctx, "/usr/bin/sleep 10", 300)
		
		// Command should be interrupted
		if !interrupted {
			t.Errorf("Expected command to be interrupted, but it wasn't. stdout: %q, stderr: %q, exitCode: %d", stdout, stderr, exitCode)
		}

		// Shell should still be alive
		if !shell.isAlive {
			t.Error("Shell should still be alive after interrupted command")
		}

		// Verify shell can still execute commands
		ctx2 := context.Background()
		stdout, stderr, exitCode, interrupted, err = shell.Exec(ctx2, "echo 'recovery test'", 5000)
		if err != nil {
			t.Fatalf("Failed to execute command after interruption: %v", err)
		}
		if exitCode != 0 {
			t.Errorf("Recovery command failed: exit code %d, stderr: %s", exitCode, stderr)
		}
		if interrupted {
			t.Error("Recovery command should not be interrupted")
		}
		if !strings.Contains(stdout, "recovery test") {
			t.Errorf("Expected 'recovery test' in output, got: %q", stdout)
		}
	})

	// Test 5: Complex commands and quoting
	t.Run("Complex Commands", func(t *testing.T) {
		// Test with safe commands that should be available
		stdout, stderr, exitCode, _, err := shell.Exec(ctx, "echo 'line1' && echo 'line2'", 5000)
		if err != nil {
			t.Fatalf("Failed to execute complex command: %v", err)
		}
		if exitCode != 0 {
			t.Errorf("Complex command failed with exit code %d, stderr: %s", exitCode, stderr)
		}
		if !strings.Contains(stdout, "line1") || !strings.Contains(stdout, "line2") {
			t.Errorf("Complex command output should contain both lines, got: %q", stdout)
		}
	})

	// Test 6: Error handling
	t.Run("Error Handling", func(t *testing.T) {
		// Execute a command that should fail
		_, _, exitCode, _, err := shell.Exec(ctx, "nonexistentcommand12345", 5000)
		if err != nil {
			t.Fatalf("Failed to execute nonexistent command: %v", err)
		}
		// Should have non-zero exit code
		if exitCode == 0 {
			t.Error("Nonexistent command should have non-zero exit code")
		}
		// Shell should still be functional
		if !shell.isAlive {
			t.Error("Shell should still be alive after failed command")
		}
	})
}

func TestUnixQuotingBehavior(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix-specific test")
	}

	tests := []struct {
		name     string
		input    string
		contains string // What the quoted result should contain
	}{
		{"simple", "hello", "'hello'"},
		{"with spaces", "hello world", "'hello world'"},
		{"empty", "", "''"},
		{"with dollar", "echo $HOME", "'echo $HOME'"}, // Should be literal, not expanded
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := shellQuote(test.input)
			if !strings.Contains(result, test.contains) {
				t.Errorf("shellQuote(%q) = %q, should contain %q", test.input, result, test.contains)
			}
		})
	}
}
