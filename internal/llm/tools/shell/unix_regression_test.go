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

func TestUnixShellQuoting(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix-specific test")
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty string", "", "''"},
		{"simple string", "hello", "'hello'"},
		{"string with spaces", "hello world", "'hello world'"},
		{"string with single quotes", "test with 'quotes'", "'test with '\\''quotes'\\''"},
		{"string with special chars", "echo $HOME && ls -la", "'echo $HOME && ls -la'"},
		{"complex command", "find . -name '*.go' | grep test", "'find . -name '\\''*.go'\\'' | grep test'"},
		{"string with pipe", "ps aux | grep bash", "'ps aux | grep bash'"},
		{"string with semicolon", "cd /tmp; ls", "'cd /tmp; ls'"},
		{"string with ampersand", "cmd1 && cmd2", "'cmd1 && cmd2'"},
		{"string with backticks", "echo `date`", "'echo `date`'"},
		{"string with dollar", "echo $PATH", "'echo $PATH'"},
		{"string with backslashes", "echo \\n\\t", "'echo \\n\\t'"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := shellQuote(test.input)
			if result != test.expected {
				t.Errorf("shellQuote(%q) = %q, want %q", test.input, result, test.expected)
			}
		})
	}
}

func TestUnixShellPersistence(t *testing.T) {
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

	// Test shell is alive
	if !shell.isAlive {
		t.Error("Shell should be alive after creation")
	}

	// Test working directory persistence
	ctx := context.Background()
	stdout, stderr, exitCode, interrupted, err := shell.Exec(ctx, "pwd", 5000)
	if err != nil {
		t.Fatalf("Failed to execute pwd: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("pwd failed with exit code %d, stderr: %s", exitCode, stderr)
	}
	if interrupted {
		t.Error("pwd command was interrupted")
	}

	// The output should contain our temp directory
	if !strings.Contains(stdout, tempDir) {
		t.Errorf("pwd output %q should contain temp dir %q", stdout, tempDir)
	}

	// Test directory change persistence
	subDir := tempDir + "/subdir"
	os.Mkdir(subDir, 0755)
	
	stdout, stderr, exitCode, interrupted, err = shell.Exec(ctx, "cd subdir", 5000)
	if err != nil {
		t.Fatalf("Failed to execute cd: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("cd failed with exit code %d, stderr: %s", exitCode, stderr)
	}

	// Verify we're in the subdirectory
	stdout, stderr, exitCode, interrupted, err = shell.Exec(ctx, "pwd", 5000)
	if err != nil {
		t.Fatalf("Failed to execute pwd after cd: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("pwd after cd failed with exit code %d, stderr: %s", exitCode, stderr)
	}
	if !strings.Contains(stdout, subDir) {
		t.Errorf("After cd, pwd output %q should contain subdir %q", stdout, subDir)
	}

	// Test environment variable persistence
	stdout, stderr, exitCode, interrupted, err = shell.Exec(ctx, "export TEST_VAR=hello", 5000)
	if err != nil {
		t.Fatalf("Failed to execute export: %v", err)
	}

	stdout, stderr, exitCode, interrupted, err = shell.Exec(ctx, "echo $TEST_VAR", 5000)
	if err != nil {
		t.Fatalf("Failed to execute echo: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("echo failed with exit code %d, stderr: %s", exitCode, stderr)
	}
	if !strings.Contains(stdout, "hello") {
		t.Errorf("Environment variable not persisted, got: %q", stdout)
	}
}

func TestUnixSignalHandling(t *testing.T) {
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

	// Test command timeout/cancellation
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Start a long-running command that should be interrupted
	stdout, stderr, exitCode, interrupted, err := shell.Exec(ctx, "sleep 5", 200)
	
	// The command should be interrupted due to context timeout
	if !interrupted {
		t.Error("Long-running command should have been interrupted")
	}
	
	// Exit code should indicate interruption (143 = SIGTERM)
	if exitCode != 143 {
		t.Errorf("Expected exit code 143 (SIGTERM), got %d", exitCode)
	}

	// Shell should still be alive after interruption
	if !shell.isAlive {
		t.Error("Shell should still be alive after command interruption")
	}

	// Test that shell can still execute commands after interruption
	ctx2 := context.Background()
	stdout, stderr, exitCode, interrupted, err = shell.Exec(ctx2, "echo 'still alive'", 5000)
	if err != nil {
		t.Fatalf("Failed to execute command after interruption: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("Command failed after interruption: exit code %d, stderr: %s", exitCode, stderr)
	}
	if interrupted {
		t.Error("Simple command should not be interrupted")
	}
	if !strings.Contains(stdout, "still alive") {
		t.Errorf("Expected 'still alive' in output, got: %q", stdout)
	}
}

func TestUnixComplexCommands(t *testing.T) {
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

	// Test pipe commands
	stdout, stderr, exitCode, _, err := shell.Exec(ctx, "echo 'line1\nline2\nline3' | grep 'line2'", 5000)
	if err != nil {
		t.Fatalf("Failed to execute pipe command: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("Pipe command failed with exit code %d, stderr: %s", exitCode, stderr)
	}
	if !strings.Contains(stdout, "line2") {
		t.Errorf("Pipe command output should contain 'line2', got: %q", stdout)
	}

	// Test command with && operator
	stdout, stderr, exitCode, _, err = shell.Exec(ctx, "echo 'first' && echo 'second'", 5000)
	if err != nil {
		t.Fatalf("Failed to execute && command: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("&& command failed with exit code %d, stderr: %s", exitCode, stderr)
	}
	if !strings.Contains(stdout, "first") || !strings.Contains(stdout, "second") {
		t.Errorf("&& command output should contain both 'first' and 'second', got: %q", stdout)
	}

	// Test command with error handling
	stdout, stderr, exitCode, _, err = shell.Exec(ctx, "echo 'before'; false; echo 'after'", 5000)
	if err != nil {
		t.Fatalf("Failed to execute semicolon command: %v", err)
	}
	// Should have non-zero exit code due to 'false' command
	if exitCode == 0 {
		t.Error("Command with 'false' should have non-zero exit code")
	}

	// Test command with variable substitution
	stdout, stderr, exitCode, _, err = shell.Exec(ctx, "VAR='hello world'; echo \"$VAR\"", 5000)
	if err != nil {
		t.Fatalf("Failed to execute variable command: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("Variable command failed with exit code %d, stderr: %s", exitCode, stderr)
	}
	if !strings.Contains(stdout, "hello world") {
		t.Errorf("Variable substitution failed, got: %q", stdout)
	}
}

func TestUnixShellDetection(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix-specific test")
	}

	kind := DetectShellKind()
	if kind != UnixBash {
		t.Errorf("On Unix systems, DetectShellKind() should return UnixBash, got %v", kind)
	}
}

func TestUnixShellDefaults(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix-specific test")
	}

	path, args := getShellDefaults(UnixBash)
	if path != "/bin/bash" {
		t.Errorf("Expected path '/bin/bash', got '%s'", path)
	}
	if len(args) != 1 || args[0] != "-l" {
		t.Errorf("Expected args ['-l'], got %v", args)
	}
}

func TestUnixGenerateWrappedCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix-specific test")
	}

	userCommand := "echo hello"
	stdoutFile := "/tmp/stdout.txt"
	stderrFile := "/tmp/stderr.txt"
	statusFile := "/tmp/status.txt"
	cwdFile := "/tmp/cwd.txt"

	result := generateWrappedCommand(UnixBash, userCommand, stdoutFile, stderrFile, statusFile, cwdFile)

	// Check that the result contains expected elements for Unix
	expectedElements := []string{
		"eval 'echo hello'",
		"EXEC_EXIT_CODE=$?",
		"'/tmp/stdout.txt'",
		"'/tmp/stderr.txt'",
		"'/tmp/status.txt'",
		"'/tmp/cwd.txt'",
	}

	for _, element := range expectedElements {
		if !strings.Contains(result, element) {
			t.Errorf("Unix wrapped command should contain '%s', got: %s", element, result)
		}
	}
}
