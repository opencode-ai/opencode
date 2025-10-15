package shell

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/opencode-ai/opencode/internal/config"
)

// ShellKind represents the type of shell being used
type ShellKind int

const (
	UnixBash ShellKind = iota
	Pwsh
	WindowsPowerShell
	CmdExe
)

// String returns the string representation of ShellKind
func (sk ShellKind) String() string {
	switch sk {
	case UnixBash:
		return "UnixBash"
	case Pwsh:
		return "Pwsh"
	case WindowsPowerShell:
		return "WindowsPowerShell"
	case CmdExe:
		return "CmdExe"
	default:
		return "Unknown"
	}
}

// DetectShellKind detects the appropriate shell kind for the current platform
func DetectShellKind() ShellKind {
	if runtime.GOOS != "windows" {
		// On Unix systems, keep existing logic returning UnixBash
		return UnixBash
	}

	// On Windows, rank availability: pwsh > powershell > cmd
	// Check for PowerShell Core (pwsh) first
	if _, err := exec.LookPath("pwsh"); err == nil {
		return Pwsh
	}

	// Check for Windows PowerShell (powershell)
	if _, err := exec.LookPath("powershell"); err == nil {
		return WindowsPowerShell
	}

	// Check for Command Prompt (cmd)
	if _, err := exec.LookPath("cmd"); err == nil {
		return CmdExe
	}

	// Fallback to cmd (should always be available on Windows)
	return CmdExe
}

// getShellDefaults returns the shell path and arguments for a given shell kind
func getShellDefaults(shellKind ShellKind) (string, []string) {
	switch shellKind {
	case Pwsh:
		return "pwsh", []string{"-NoLogo", "-NoExit", "-Command", "-"}
	case WindowsPowerShell:
		return "powershell", []string{"-NoLogo", "-NoExit", "-Command", "-"}
	case CmdExe:
		return "cmd.exe", []string{"/Q", "/K"}
	default:
		// Unix bash or fallback
		return "/bin/bash", []string{"-l"}
	}
}

type PersistentShell struct {
	cmd          *exec.Cmd
	stdin        *os.File
	isAlive      bool
	cwd          string
	mu           sync.Mutex
	commandQueue chan *commandExecution
}

type commandExecution struct {
	command    string
	timeout    time.Duration
	resultChan chan commandResult
	ctx        context.Context
}

type commandResult struct {
	stdout      string
	stderr      string
	exitCode    int
	interrupted bool
	err         error
}

var (
	shellInstance     *PersistentShell
	shellInstanceOnce sync.Once
)

func GetPersistentShell(workingDir string) *PersistentShell {
	shellInstanceOnce.Do(func() {
		shellInstance = newPersistentShell(workingDir)
	})

	if shellInstance == nil {
		shellInstance = newPersistentShell(workingDir)
		if shellInstance == nil {
			panic("Failed to create persistent shell: unable to start shell process")
		}
	} else if !shellInstance.isAlive {
		newShell := newPersistentShell(shellInstance.cwd)
		if newShell == nil {
			panic("Failed to recreate persistent shell: unable to start shell process")
		}
		shellInstance = newShell
	}

	return shellInstance
}

// buildExecCommand creates an exec.Cmd for the specified shell kind with given path and args
func buildExecCommand(kind ShellKind, path string, args []string) *exec.Cmd {
	return exec.Command(path, args...)
}

// injectEnvironment applies platform-specific environment configuration to the command
func injectEnvironment(cmd *exec.Cmd) {
	cmd.Env = append(os.Environ(), "GIT_EDITOR=true")
}

func newPersistentShell(cwd string) *PersistentShell {
	// Get shell configuration from config
	cfg := config.Get()
	
	// Default to config values if available
	var shellPath string
	var shellArgs []string
	var shellKind ShellKind
	
	if cfg != nil {
		shellPath = cfg.Shell.Path
		shellArgs = cfg.Shell.Args
	}
	
	// Fallback logic if config is empty
	if shellPath == "" {
		if runtime.GOOS == "windows" {
			// On Windows, use detected shell kind for fallback
			shellKind = DetectShellKind()
			shellPath, shellArgs = getShellDefaults(shellKind)
		} else {
			// On Unix, use traditional approach
			shellKind = UnixBash
			shellPath = os.Getenv("SHELL")
			if shellPath == "" {
				shellPath = "/bin/bash"
			}
			if len(shellArgs) == 0 {
				shellArgs = []string{"-l"}
			}
		}
	} else {
		// Determine shell kind from configured path for consistency
		if runtime.GOOS == "windows" {
			shellKind = DetectShellKind()
		} else {
			shellKind = UnixBash
		}
	}

	cmd := buildExecCommand(shellKind, shellPath, shellArgs)
	cmd.Dir = cwd

	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		return nil
	}

	injectEnvironment(cmd)

	err = cmd.Start()
	if err != nil {
		return nil
	}

	shell := &PersistentShell{
		cmd:          cmd,
		stdin:        stdinPipe.(*os.File),
		isAlive:      true,
		cwd:          cwd,
		commandQueue: make(chan *commandExecution, 10),
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Fprintf(os.Stderr, "Panic in shell command processor: %v\n", r)
				shell.isAlive = false
				close(shell.commandQueue)
			}
		}()
		shell.processCommands()
	}()

	go func() {
		err := cmd.Wait()
		if err != nil {
			// Log the error if needed
		}
		shell.isAlive = false
		close(shell.commandQueue)
	}()

	return shell
}

func (s *PersistentShell) processCommands() {
	for cmd := range s.commandQueue {
		result := s.execCommand(cmd.command, cmd.timeout, cmd.ctx)
		cmd.resultChan <- result
	}
}

func (s *PersistentShell) execCommand(command string, timeout time.Duration, ctx context.Context) commandResult {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isAlive {
		return commandResult{
			stderr:   "Shell is not alive",
			exitCode: 1,
			err:      errors.New("shell is not alive"),
		}
	}

	// Create temporary files using os.CreateTemp to avoid manual path concatenation
	stdoutFile, err := os.CreateTemp("", "opencode-stdout-*")
	if err != nil {
		return commandResult{
			stderr:   fmt.Sprintf("Failed to create stdout temp file: %v", err),
			exitCode: 1,
			err:      err,
		}
	}
	stdoutFile.Close()

	stderrFile, err := os.CreateTemp("", "opencode-stderr-*")
	if err != nil {
		os.Remove(stdoutFile.Name())
		return commandResult{
			stderr:   fmt.Sprintf("Failed to create stderr temp file: %v", err),
			exitCode: 1,
			err:      err,
		}
	}
	stderrFile.Close()

	statusFile, err := os.CreateTemp("", "opencode-status-*")
	if err != nil {
		os.Remove(stdoutFile.Name())
		os.Remove(stderrFile.Name())
		return commandResult{
			stderr:   fmt.Sprintf("Failed to create status temp file: %v", err),
			exitCode: 1,
			err:      err,
		}
	}
	statusFile.Close()

	cwdFile, err := os.CreateTemp("", "opencode-cwd-*")
	if err != nil {
		os.Remove(stdoutFile.Name())
		os.Remove(stderrFile.Name())
		os.Remove(statusFile.Name())
		return commandResult{
			stderr:   fmt.Sprintf("Failed to create cwd temp file: %v", err),
			exitCode: 1,
			err:      err,
		}
	}
	cwdFile.Close()

	defer func() {
		os.Remove(stdoutFile.Name())
		os.Remove(stderrFile.Name())
		os.Remove(statusFile.Name())
		os.Remove(cwdFile.Name())
	}()

	// Detect shell kind for proper command wrapping
	var shellKind ShellKind
	if runtime.GOOS == "windows" {
		cfg := config.Get()
		if cfg != nil && cfg.Shell.Path != "" {
			// Determine shell kind from configured path
			if strings.Contains(strings.ToLower(cfg.Shell.Path), "cmd") {
				shellKind = CmdExe
			} else if strings.Contains(strings.ToLower(cfg.Shell.Path), "pwsh") {
				shellKind = Pwsh
			} else if strings.Contains(strings.ToLower(cfg.Shell.Path), "powershell") {
				shellKind = WindowsPowerShell
			} else {
				// Fallback to detection
				shellKind = DetectShellKind()
			}
		} else {
			// Fallback detection
			shellKind = DetectShellKind()
		}
	} else {
		shellKind = UnixBash
	}

	// Generate the wrapped command using the new function
	fullCommand := generateWrappedCommand(shellKind, command, stdoutFile.Name(), stderrFile.Name(), statusFile.Name(), cwdFile.Name())

	_, err = s.stdin.Write([]byte(fullCommand + "\n"))
	if err != nil {
		return commandResult{
			stderr:   fmt.Sprintf("Failed to write command to shell: %v", err),
			exitCode: 1,
			err:      err,
		}
	}

	interrupted := false

	startTime := time.Now()

	done := make(chan bool)
	go func() {
		for {
			select {
			case <-ctx.Done():
				s.killChildren()
				interrupted = true
				done <- true
				return

			case <-time.After(10 * time.Millisecond):
				if fileExists(statusFile.Name()) && fileSize(statusFile.Name()) > 0 {
					done <- true
					return
				}

				if timeout > 0 {
					elapsed := time.Since(startTime)
					if elapsed > timeout {
						s.killChildren()
						interrupted = true
						done <- true
						return
					}
				}
			}
		}
	}()

	<-done

	stdout := readFileOrEmpty(stdoutFile.Name())
	stderr := readFileOrEmpty(stderrFile.Name())
	exitCodeStr := readFileOrEmpty(statusFile.Name())
	newCwd := readFileOrEmpty(cwdFile.Name())

	exitCode := 0
	if exitCodeStr != "" {
		fmt.Sscanf(exitCodeStr, "%d", &exitCode)
	} else if interrupted {
		exitCode = 143
		stderr += "\nCommand execution timed out or was interrupted"
	}

	if newCwd != "" {
		s.cwd = strings.TrimSpace(newCwd)
	}

	return commandResult{
		stdout:      stdout,
		stderr:      stderr,
		exitCode:    exitCode,
		interrupted: interrupted,
	}
}


func (s *PersistentShell) Exec(ctx context.Context, command string, timeoutMs int) (string, string, int, bool, error) {
	if !s.isAlive {
		return "", "Shell is not alive", 1, false, errors.New("shell is not alive")
	}

	timeout := time.Duration(timeoutMs) * time.Millisecond

	resultChan := make(chan commandResult)
	s.commandQueue <- &commandExecution{
		command:    command,
		timeout:    timeout,
		resultChan: resultChan,
		ctx:        ctx,
	}

	result := <-resultChan
	return result.stdout, result.stderr, result.exitCode, result.interrupted, result.err
}

func (s *PersistentShell) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isAlive {
		return
	}

	s.stdin.Write([]byte("exit\n"))

	s.cmd.Process.Kill()
	s.isAlive = false
}

// shellQuote quotes a string for Unix bash shell
// Uses single quotes for simplicity but handles embedded single quotes properly
func shellQuote(s string) string {
	// If the string is empty, return empty quotes
	if s == "" {
		return "''"
	}
	
	// If the string contains no special characters, we can use simple single quotes
	if !containsSpecialChars(s) {
		return "'" + s + "'"
	}
	
	// For strings with special characters, use proper escaping
	// Replace single quotes with '\''
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// shellQuoteWindows quotes a string for Windows shells based on shell kind
// Handles complex commands with &, |, ; and other special characters
func shellQuoteWindows(kind ShellKind, s string) string {
	switch kind {
	case CmdExe:
		return shellQuoteCmd(s)
	case Pwsh, WindowsPowerShell:
		return shellQuotePowerShell(s)
	default:
		// Fallback to CMD behavior
		return shellQuoteCmd(s)
	}
}

// shellQuoteCmd quotes a string for Windows CMD
// CMD has complex quoting rules, especially for special characters
func shellQuoteCmd(s string) string {
	if s == "" {
		return `""`
	}
	
	// Check if we need quoting
	needsQuoting := containsCmdSpecialChars(s)
	
	if !needsQuoting {
		// Simple case - no special characters
		return s
	}
	
	// Complex case - escape and quote
	escaped := s
	
	// Escape double quotes by doubling them
	escaped = strings.ReplaceAll(escaped, `"`, `""`)
	
	// Handle special characters that need escaping in CMD
	// Note: & | < > ^ need to be inside quotes to be literal
	
	return `"` + escaped + `"`
}

// shellQuotePowerShell quotes a string for PowerShell (pwsh and powershell)
// PowerShell has different quoting rules than CMD
func shellQuotePowerShell(s string) string {
	if s == "" {
		return `""`
	}
	
	// Check if we need quoting
	needsQuoting := containsPowerShellSpecialChars(s)
	
	if !needsQuoting {
		// Simple case - no special characters
		return s
	}
	
	// For PowerShell, we can use single quotes for most cases
	// as they preserve literal strings better
	if !strings.Contains(s, "'") {
		return "'" + s + "'"
	}
	
	// If string contains single quotes, use double quotes and escape
	escaped := s
	
	// Escape backticks (PowerShell escape character)
	escaped = strings.ReplaceAll(escaped, "`", "``")
	
	// Escape double quotes
	escaped = strings.ReplaceAll(escaped, `"`, "`\"")
	
	// Escape dollar signs (variable expansion)
	escaped = strings.ReplaceAll(escaped, "$", "`$")
	
	return `"` + escaped + `"`
}

// containsSpecialChars checks if a string contains characters that need quoting in bash
func containsSpecialChars(s string) bool {
	specialChars := " \t\n\r'\"\\|&;<>(){}[]$`*?~#"
	for _, char := range s {
		if strings.ContainsRune(specialChars, char) {
			return true
		}
	}
	return false
}

// containsCmdSpecialChars checks if a string contains characters that need quoting in CMD
func containsCmdSpecialChars(s string) bool {
	specialChars := " \t\n\r\"&|<>^()%!"
	for _, char := range s {
		if strings.ContainsRune(specialChars, char) {
			return true
		}
	}
	return false
}

// containsPowerShellSpecialChars checks if a string contains characters that need quoting in PowerShell
func containsPowerShellSpecialChars(s string) bool {
	specialChars := " \t\n\r'\"\\|&;<>(){}[]$`*?~#@"
	for _, char := range s {
		if strings.ContainsRune(specialChars, char) {
			return true
		}
	}
	return false
}

// generateWrappedCommand creates a heredoc command wrapper for the specified shell kind
func generateWrappedCommand(kind ShellKind, userCommand string, stdoutFile, stderrFile, statusFile, cwdFile string) string {
	switch kind {
	case CmdExe:
		// CMD syntax with proper redirection and status capture
		return fmt.Sprintf("%s >%s 2>%s\necho %%ERRORLEVEL%% >%s\ncd >%s\n",
			userCommand,
			shellQuoteWindows(kind, stdoutFile),
			shellQuoteWindows(kind, stderrFile),
			shellQuoteWindows(kind, statusFile),
			shellQuoteWindows(kind, cwdFile),
		)
	case Pwsh, WindowsPowerShell:
		// PowerShell command to execute the user's command, redirect streams, and capture exit code and CWD.
		// 1. Execute the user command, redirecting stdout and stderr to their respective files.
		// 2. Check the success status ($?). If true, exit code is 0.
		// 3. If false, check $LASTEXITCODE for native executables. If it's non-zero, use it.
		// 4. Otherwise, for failed cmdlets, default to exit code 1.
		// 5. Write the determined exit code to the status file.
		// 6. Write the current directory path to the CWD file.
		psExitCodeLogic := fmt.Sprintf(
			"if ($?) { '0' } else { if ($LASTEXITCODE -ne 0) { $LASTEXITCODE } else { '1' } } | Out-File -FilePath %s -Encoding utf8",
			shellQuoteWindows(kind, statusFile),
		)
		psCwdLogic := fmt.Sprintf(
			"(Get-Location).Path | Out-File -FilePath %s -Encoding utf8",
			shellQuoteWindows(kind, cwdFile),
		)
		return fmt.Sprintf("%s 1>%s 2>%s; %s; %s\n",
			userCommand,
			shellQuoteWindows(kind, stdoutFile),
			shellQuoteWindows(kind, stderrFile),
			psExitCodeLogic,
			psCwdLogic,
		)
	default:
		// Unix bash fallback
		return fmt.Sprintf("\neval %s </dev/null >%s 2>%s\nEXEC_EXIT_CODE=$?\npwd >%s\necho $EXEC_EXIT_CODE >%s\n",
			shellQuote(userCommand),
			shellQuote(stdoutFile),
			shellQuote(stderrFile),
			shellQuote(cwdFile),
			shellQuote(statusFile),
		)
	}
}

func readFileOrEmpty(path string) string {
	content, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(content)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func fileSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}
