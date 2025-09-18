package util

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

// FocusTracker manages terminal focus tracking using ANSI escape sequences
type FocusTracker struct {
	program *tea.Program
	focused bool
}

// NewFocusTracker creates a new focus tracker
func NewFocusTracker(program *tea.Program) *FocusTracker {
	return &FocusTracker{
		program: program,
		focused: true, // Default to focused
	}
}

// Start enables focus tracking and starts monitoring
func (ft *FocusTracker) Start(ctx context.Context) error {
	// Enable focus tracking with ANSI escape sequence
	fmt.Print("\x1b[?1004h")

	// Start a goroutine to handle focus events
	go func() {
		<-ctx.Done()
		// Disable focus tracking when context is cancelled
		fmt.Print("\x1b[?1004l")
	}()

	return nil
}

// HandleFocusEvent processes focus in/out events from terminal
func (ft *FocusTracker) HandleFocusEvent(focused bool) {
	if ft.focused != focused {
		ft.focused = focused
		if ft.program != nil {
			ft.program.Send(FocusMsg{Focused: focused})
		}
	}
}

// IsFocused returns the current focus state
func (ft *FocusTracker) IsFocused() bool {
	return ft.focused
}

// ParseFocusMessage takes an input key event and checks if it matches
// the ANSI escape codes for focus in or out.
func ParseFocusMessage(input tea.KeyMsg) (bool, FocusMsg) {
	switch input.String() {
	case "\x1b[I": // Focus in
		return true, FocusMsg{Focused: true}
	case "\x1b[O": // Focus out
		return true, FocusMsg{Focused: false}
	}
	return false, FocusMsg{}
}
