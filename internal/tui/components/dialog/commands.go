package dialog

import (
	"github.com/charmbracelet/bubbles/v2/key"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/kujtimiihoxha/opencode/internal/tui/layout"
	"github.com/kujtimiihoxha/opencode/internal/tui/styles"
	"github.com/kujtimiihoxha/opencode/internal/tui/util"
)

// Command represents a command that can be executed
type Command struct {
	ID          string
	Title       string
	Description string
	Handler     func(cmd Command) tea.Cmd
}

// CommandSelectedMsg is sent when a command is selected
type CommandSelectedMsg struct {
	Command Command
}

// CloseCommandDialogMsg is sent when the command dialog is closed
type CloseCommandDialogMsg struct{}

// CommandDialog interface for the command selection dialog
type CommandDialog interface {
	layout.ModelWithView
	layout.Bindings
	SetCommands(commands []Command)
	SetSelectedCommand(commandID string)
}

type commandDialogCmp struct {
	commands          []Command
	selectedIdx       int
	width             int
	height            int
	selectedCommandID string
}

type commandKeyMap struct {
	Up     key.Binding
	Down   key.Binding
	Enter  key.Binding
	Escape key.Binding
	J      key.Binding
	K      key.Binding
}

var commandKeys = commandKeyMap{
	Up: key.NewBinding(
		key.WithKeys("up"),
		key.WithHelp("↑", "previous command"),
	),
	Down: key.NewBinding(
		key.WithKeys("down"),
		key.WithHelp("↓", "next command"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select command"),
	),
	Escape: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "close"),
	),
	J: key.NewBinding(
		key.WithKeys("j"),
		key.WithHelp("j", "next command"),
	),
	K: key.NewBinding(
		key.WithKeys("k"),
		key.WithHelp("k", "previous command"),
	),
}

func (c *commandDialogCmp) Init() tea.Cmd {
	return nil
}

func (c *commandDialogCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, commandKeys.Up) || key.Matches(msg, commandKeys.K):
			if c.selectedIdx > 0 {
				c.selectedIdx--
			}
			return c, nil
		case key.Matches(msg, commandKeys.Down) || key.Matches(msg, commandKeys.J):
			if c.selectedIdx < len(c.commands)-1 {
				c.selectedIdx++
			}
			return c, nil
		case key.Matches(msg, commandKeys.Enter):
			if len(c.commands) > 0 {
				return c, util.CmdHandler(CommandSelectedMsg{
					Command: c.commands[c.selectedIdx],
				})
			}
		case key.Matches(msg, commandKeys.Escape):
			return c, util.CmdHandler(CloseCommandDialogMsg{})
		}
	case tea.WindowSizeMsg:
		c.width = msg.Width
		c.height = msg.Height
	}
	return c, nil
}

func (c *commandDialogCmp) View() string {
	if len(c.commands) == 0 {
		return styles.BaseStyle.Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderBackground(styles.Background).
			BorderForeground(styles.ForgroundDim).
			Width(40).
			Render("No commands available")
	}

	// Calculate max width needed for command titles
	maxWidth := 40 // Minimum width
	for _, cmd := range c.commands {
		if len(cmd.Title) > maxWidth-4 { // Account for padding
			maxWidth = len(cmd.Title) + 4
		}
		if len(cmd.Description) > maxWidth-4 {
			maxWidth = len(cmd.Description) + 4
		}
	}

	// Limit height to avoid taking up too much screen space
	maxVisibleCommands := min(10, len(c.commands))

	// Build the command list
	commandItems := make([]string, 0, maxVisibleCommands)
	startIdx := 0

	// If we have more commands than can be displayed, adjust the start index
	if len(c.commands) > maxVisibleCommands {
		// Center the selected item when possible
		halfVisible := maxVisibleCommands / 2
		if c.selectedIdx >= halfVisible && c.selectedIdx < len(c.commands)-halfVisible {
			startIdx = c.selectedIdx - halfVisible
		} else if c.selectedIdx >= len(c.commands)-halfVisible {
			startIdx = len(c.commands) - maxVisibleCommands
		}
	}

	endIdx := min(startIdx+maxVisibleCommands, len(c.commands))

	for i := startIdx; i < endIdx; i++ {
		cmd := c.commands[i]
		itemStyle := styles.BaseStyle.Width(maxWidth)
		descStyle := styles.BaseStyle.Width(maxWidth).Foreground(styles.ForgroundDim)

		if i == c.selectedIdx {
			itemStyle = itemStyle.
				Background(styles.PrimaryColor).
				Foreground(styles.Background).
				Bold(true)
			descStyle = descStyle.
				Background(styles.PrimaryColor).
				Foreground(styles.Background)
		}

		title := itemStyle.Padding(0, 1).Render(cmd.Title)
		description := ""
		if cmd.Description != "" {
			description = descStyle.Padding(0, 1).Render(cmd.Description)
			commandItems = append(commandItems, lipgloss.JoinVertical(lipgloss.Left, title, description))
		} else {
			commandItems = append(commandItems, title)
		}
	}

	title := styles.BaseStyle.
		Foreground(styles.PrimaryColor).
		Bold(true).
		Width(maxWidth).
		Padding(0, 1).
		Render("Commands")

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		styles.BaseStyle.Width(maxWidth).Render(""),
		styles.BaseStyle.Width(maxWidth).Render(lipgloss.JoinVertical(lipgloss.Left, commandItems...)),
		styles.BaseStyle.Width(maxWidth).Render(""),
	)

	return styles.BaseStyle.Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderBackground(styles.Background).
		BorderForeground(styles.ForgroundDim).
		Width(lipgloss.Width(content) + 4).
		Render(content)
}

func (c *commandDialogCmp) BindingKeys() []key.Binding {
	return layout.KeyMapToSlice(commandKeys)
}

func (c *commandDialogCmp) SetCommands(commands []Command) {
	c.commands = commands

	// If we have a selected command ID, find its index
	if c.selectedCommandID != "" {
		for i, cmd := range commands {
			if cmd.ID == c.selectedCommandID {
				c.selectedIdx = i
				return
			}
		}
	}

	// Default to first command if selected not found
	c.selectedIdx = 0
}

func (c *commandDialogCmp) SetSelectedCommand(commandID string) {
	c.selectedCommandID = commandID

	// Update the selected index if commands are already loaded
	if len(c.commands) > 0 {
		for i, cmd := range c.commands {
			if cmd.ID == commandID {
				c.selectedIdx = i
				return
			}
		}
	}
}

// NewCommandDialogCmp creates a new command selection dialog
func NewCommandDialogCmp() CommandDialog {
	return &commandDialogCmp{
		commands:          []Command{},
		selectedIdx:       0,
		selectedCommandID: "",
	}
}
