# Windows-Specific Shell Tests

This document describes the Windows-specific tests that have been created for the shell execution functionality.

## Test Files

### 1. `shell_windows_test.go`
Comprehensive test suite with Windows build tags (`//go:build windows`) that covers:

#### Test Functions

**TestStdoutCaptureWindows**
- Tests stdout capture for various Windows shells (CMD, PowerShell)
- Verifies commands like `echo`, `dir`, and PowerShell `Write-Output`
- Ensures output is properly captured and contains expected content

**TestStderrCaptureWindows**
- Tests stderr capture for error conditions
- Tests non-existent commands, invalid directories, PowerShell errors
- Verifies proper error handling and exit codes

**TestTimeoutWindows**
- Tests command timeout behavior using:
  - `timeout /t 3 /nobreak` (CMD)
  - `Start-Sleep -Seconds 3` (PowerShell)
  - `ping -n 5 127.0.0.1` (network timeout)
- Verifies interruption occurs within expected timeframes
- Checks exit codes for timed-out processes

**TestInterruptionWindows**
- Tests context cancellation behavior
- Uses goroutine to cancel context after 500ms
- Verifies quick interruption response
- Tests `killChildren()` functionality

**TestCwdUpdatesWindows**
- Tests working directory changes using `cd /d`
- Creates temporary directory for testing
- Verifies shell tracks current working directory changes
- Tests file creation in changed directory

**TestKillChildrenWindows**
- Tests the Windows-specific `killChildren()` implementation
- Starts multiple child processes using `&` operator
- Verifies processes are properly terminated
- Tests shell remains alive after killing children
- Verifies subsequent commands still work

**TestPowerShellSpecificWindows**
- Tests PowerShell-specific features:
  - `Get-Process` commands
  - Variable assignment and usage
  - Error handling with try/catch blocks
- Validates PowerShell command execution and output

**TestCmdSpecificWindows**
- Tests CMD-specific features:
  - Environment variable setting and expansion
  - Pipe operations with `findstr`
  - Custom exit codes with `exit /b`
- Validates CMD command execution and behavior

### 2. `simple_windows_test.go`
Basic demonstration tests:

**TestSimpleWindowsExecution**
- Basic echo command test
- Logs all execution details for debugging
- Validates basic shell functionality

**TestWindowsShellDetection**
- Tests the `DetectShellKind()` function on Windows
- Verifies correct detection of Windows shells (pwsh, powershell, cmd)

## Windows Shell Support

The tests cover three Windows shell types:

1. **PowerShell Core (pwsh)** - Modern cross-platform PowerShell
2. **Windows PowerShell (powershell)** - Traditional Windows PowerShell
3. **Command Prompt (cmd.exe)** - Traditional Windows command line

## Key Features Tested

### 1. Stdout/Stderr Capture
- Proper redirection using temporary files
- Cross-shell compatibility (CMD vs PowerShell syntax)
- Multi-line output handling
- Error output separation

### 2. Timeout Handling
- Internal timeout mechanism (timeoutMs parameter)
- Context-based cancellation
- Process tree termination
- Proper cleanup of hanging processes

### 3. Interruption Handling
- Context cancellation propagation
- `killChildren()` Windows implementation
- Console control events (CTRL_BREAK_EVENT)
- Graceful vs forceful termination

### 4. Working Directory Updates
- Cross-command directory persistence
- Windows path handling
- Drive changes with `cd /d`
- Validation through file operations

### 5. Kill Children Behavior
- Windows process tree termination
- `taskkill /F /T /PID` usage
- Console control event generation
- Fallback to `Process.Kill()`

## Windows-Specific Implementation Details

### Process Termination (`kill_windows.go`)
```go
// Uses taskkill for comprehensive process tree termination
taskkillCmd := exec.Command("taskkill", "/F", "/T", "/PID", fmt.Sprintf("%d", s.cmd.Process.Pid))

// Sends console control events for graceful termination
generateConsoleCtrlEvent.Call(uintptr(CTRL_BREAK_EVENT), uintptr(0))
```

### Command Wrapping
- **CMD**: Uses `%ERRORLEVEL%` for exit code capture
- **PowerShell**: Uses `$LASTEXITCODE` and `Out-File` for redirection
- **Path quoting**: Different quoting rules for each shell type

## Running the Tests

**Note**: The current Go environment is configured for Linux (`GOOS=linux`), so these tests will not run in the current setup. To run these tests on Windows:

1. Set the Go environment for Windows:
   ```cmd
   set GOOS=windows
   set GOARCH=amd64
   ```

2. Run the Windows-specific tests:
   ```cmd
   go test -v ./internal/llm/tools/shell -run "Windows"
   ```

3. Run individual test functions:
   ```cmd
   go test -v ./internal/llm/tools/shell -run "TestStdoutCaptureWindows"
   ```

## Test Coverage

The tests provide comprehensive coverage of:
- ✅ stdout/stderr capture verification
- ✅ timeout behavior testing
- ✅ interruption and cancellation handling
- ✅ working directory updates
- ✅ killChildren functionality
- ✅ Shell-specific command syntax (CMD vs PowerShell)
- ✅ Error handling and exit codes
- ✅ Process management and cleanup

These tests ensure the shell execution functionality works correctly across all supported Windows shell environments.
