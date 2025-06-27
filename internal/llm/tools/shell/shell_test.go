package shell

import (
	"runtime"
	"strings"
	"testing"
)

func TestShellKindString(t *testing.T) {
	tests := []struct {
		kind     ShellKind
		expected string
	}{
		{UnixBash, "UnixBash"},
		{Pwsh, "Pwsh"},
		{WindowsPowerShell, "WindowsPowerShell"},
		{CmdExe, "CmdExe"},
		{ShellKind(999), "Unknown"},
	}

	for _, test := range tests {
		if got := test.kind.String(); got != test.expected {
			t.Errorf("ShellKind.String() = %v, want %v", got, test.expected)
		}
	}
}

func TestDetectShellKind(t *testing.T) {
	kind := DetectShellKind()

	if runtime.GOOS == "windows" {
		// On Windows, should detect one of the Windows shells
		switch kind {
		case Pwsh, WindowsPowerShell, CmdExe:
			// Expected
		default:
			t.Errorf("DetectShellKind() on Windows returned %v, expected a Windows shell", kind)
		}
	} else {
		// On Unix systems, should always return UnixBash
		if kind != UnixBash {
			t.Errorf("DetectShellKind() on Unix returned %v, expected UnixBash", kind)
		}
	}
}

func TestDetectShellKindValidValues(t *testing.T) {
	kind := DetectShellKind()
	
	// Ensure the returned value is one of the valid ShellKind values
	switch kind {
	case UnixBash, Pwsh, WindowsPowerShell, CmdExe:
		// Valid
	default:
		t.Errorf("DetectShellKind() returned invalid ShellKind: %v", kind)
	}
}

func TestShellQuoteWindows(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		shell    ShellKind
		expected string
	}{
		// Basic quoting tests
		{"CMD simple", "hello", CmdExe, "hello"},
		{"CMD with spaces", "hello world", CmdExe, `"hello world"`},
		{"CMD with quotes", `test with "quotes"`, CmdExe, `"test with ""quotes"""`},
		{"PowerShell simple", "hello", Pwsh, "hello"},
		{"PowerShell with spaces", "hello world", Pwsh, "'hello world'"},
		{"PowerShell with single quotes", "test with 'quotes'", Pwsh, `"test with ` + "`" + `'quotes` + "`" + `'"`},
		
		// Complex commands with special characters
		{"CMD with pipe", "echo hello | grep world", CmdExe, `"echo hello | grep world"`},
		{"CMD with ampersand", "cmd1 & cmd2", CmdExe, `"cmd1 & cmd2"`},
		{"CMD with semicolon", "cmd1; cmd2", CmdExe, `"cmd1; cmd2"`},
		{"PowerShell with pipe", "Get-Process | Where-Object Name", Pwsh, "'Get-Process | Where-Object Name'"},
		{"PowerShell with ampersand", "cmd1 & cmd2", Pwsh, "'cmd1 & cmd2'"},
		{"PowerShell with semicolon", "cmd1; cmd2", Pwsh, "'cmd1; cmd2'"},
		
		// Edge cases
		{"CMD empty", "", CmdExe, `""`},
		{"PowerShell empty", "", Pwsh, `""`},
		{"PowerShell with variables", "echo $var", Pwsh, `"echo ` + "`" + `$var"`},
		{"PowerShell with backticks", "echo `test`", Pwsh, `"echo ` + "`" + "`" + "`" + `test` + "`" + "`" + "`" + `"`},
	}
	
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := shellQuoteWindows(test.shell, test.input)
			if result != test.expected {
				t.Errorf("%s: expected %q, got %q", test.name, test.expected, result)
			}
		})
	}
}

func TestGenerateWrappedCommand(t *testing.T) {
	userCommand := "echo hello"
	stdoutFile := "C:\\temp\\stdout.txt"
	stderrFile := "C:\\temp\\stderr.txt"
	statusFile := "C:\\temp\\status.txt"
	cwdFile := "C:\\temp\\cwd.txt"
	
	// Test CMD command generation
	cmdResult := generateWrappedCommand(CmdExe, userCommand, stdoutFile, stderrFile, statusFile, cwdFile)
	if !strings.Contains(cmdResult, "echo hello >") {
		t.Errorf("CMD command generation failed: %s", cmdResult)
	}
	if !strings.Contains(cmdResult, "%ERRORLEVEL%") {
		t.Errorf("CMD command missing ERRORLEVEL capture: %s", cmdResult)
	}
	
	// Test PowerShell command generation
	pwshResult := generateWrappedCommand(Pwsh, userCommand, stdoutFile, stderrFile, statusFile, cwdFile)
	if !strings.Contains(pwshResult, "& { echo hello }") {
		t.Errorf("PowerShell command generation failed: %s", pwshResult)
	}
	if !strings.Contains(pwshResult, "$LASTEXITCODE") {
		t.Errorf("PowerShell command missing LASTEXITCODE capture: %s", pwshResult)
	}
	
	// Test Unix bash command generation
	bashResult := generateWrappedCommand(UnixBash, userCommand, stdoutFile, stderrFile, statusFile, cwdFile)
	if !strings.Contains(bashResult, "eval 'echo hello'") {
		t.Errorf("Bash command generation failed: %s", bashResult)
	}
	if !strings.Contains(bashResult, "EXEC_EXIT_CODE=$?") {
		t.Errorf("Bash command missing exit code capture: %s", bashResult)
	}
}
