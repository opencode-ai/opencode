package shell

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPersistentShell_NilSafety(t *testing.T) {
	t.Run("nil shell instance should not panic", func(t *testing.T) {
		var shell *PersistentShell = nil
		
		// Test Exec with nil shell
		stdout, stderr, exitCode, interrupted, err := shell.Exec(context.Background(), "echo test", 1000)
		
		assert.Equal(t, "", stdout)
		assert.Equal(t, "Shell instance is nil", stderr)
		assert.Equal(t, 1, exitCode)
		assert.False(t, interrupted)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "shell instance is nil")
	})

	t.Run("nil shell close should not panic", func(t *testing.T) {
		var shell *PersistentShell = nil
		
		// This should not panic
		assert.NotPanics(t, func() {
			shell.Close()
		})
	})

	t.Run("shell with nil commandQueue should not panic", func(t *testing.T) {
		shell := &PersistentShell{
			isAlive:      true,
			cwd:          "/tmp",
			commandQueue: nil, // Explicitly nil
		}
		
		stdout, stderr, exitCode, interrupted, err := shell.Exec(context.Background(), "echo test", 1000)
		
		assert.Equal(t, "", stdout)
		assert.Equal(t, "Shell command queue is not initialized", stderr)
		assert.Equal(t, 1, exitCode)
		assert.False(t, interrupted)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "shell command queue is not initialized")
	})

	t.Run("shell with isAlive false should return error", func(t *testing.T) {
		shell := &PersistentShell{
			isAlive: false,
			cwd:     "/tmp",
		}
		
		stdout, stderr, exitCode, interrupted, err := shell.Exec(context.Background(), "echo test", 1000)
		
		assert.Equal(t, "", stdout)
		assert.Equal(t, "Shell is not alive", stderr)
		assert.Equal(t, 1, exitCode)
		assert.False(t, interrupted)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "shell is not alive")
	})
}

func TestGetPersistentShell_FailureHandling(t *testing.T) {
	t.Run("should return disabled shell when creation fails", func(t *testing.T) {
		// This test is tricky because we can't easily force newPersistentShell to fail
		// But we can test that GetPersistentShell returns a non-nil shell
		shell := GetPersistentShell("/tmp")
		
		require.NotNil(t, shell)
		
		// The shell should either be alive or disabled, but not nil
		if !shell.isAlive {
			// If shell is not alive, it should handle commands gracefully
			stdout, stderr, exitCode, interrupted, err := shell.Exec(context.Background(), "echo test", 1000)
			
			assert.Equal(t, "", stdout)
			assert.Equal(t, "Shell is not alive", stderr)
			assert.Equal(t, 1, exitCode)
			assert.False(t, interrupted)
			assert.Error(t, err)
		}
	})
}

func TestShellQuote(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "'simple'"},
		{"with spaces", "'with spaces'"},
		{"with'quote", "'with'\\''quote'"},
		{"", "''"},
		{"multiple'quotes'here", "'multiple'\\''quotes'\\''here'"},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			result := shellQuote(test.input)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestFileHelpers(t *testing.T) {
	t.Run("fileExists should handle non-existent files", func(t *testing.T) {
		exists := fileExists("/non/existent/file")
		assert.False(t, exists)
	})

	t.Run("fileSize should handle non-existent files", func(t *testing.T) {
		size := fileSize("/non/existent/file")
		assert.Equal(t, int64(0), size)
	})

	t.Run("readFileOrEmpty should handle non-existent files", func(t *testing.T) {
		content := readFileOrEmpty("/non/existent/file")
		assert.Equal(t, "", content)
	})
}
