package utilComponents

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/opencode-ai/opencode/internal/tui/layout"
	"github.com/opencode-ai/opencode/internal/tui/styles"
	"github.com/opencode-ai/opencode/internal/tui/theme"
)

type SimpleListItem interface {
	Render(selected bool, width int) string
}

type SimpleList[T SimpleListItem] interface {
	tea.Model
	layout.Bindings
	GetSelectedItem() (item T, idx int)
	SetItems(items []T)
}

type simpleListCmp[T SimpleListItem] struct {
	items       []T
	selectedIdx int
	width       int
	height      int
}

type simpleListKeyMap struct {
	Up    key.Binding
	Down  key.Binding
	Enter key.Binding
}

var simpleListKeys = simpleListKeyMap{
	Up: key.NewBinding(
		key.WithKeys("up"),
		key.WithHelp("↑", "previous list item"),
	),
	Down: key.NewBinding(
		key.WithKeys("down"),
		key.WithHelp("↓", "next list item"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select item"),
	),
}

func (c *simpleListCmp[T]) Init() tea.Cmd {
	return nil
}

func (c *simpleListCmp[T]) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, simpleListKeys.Up):
			if c.selectedIdx > 0 {
				c.selectedIdx--
			}
			return c, nil
		case key.Matches(msg, simpleListKeys.Down):
			if c.selectedIdx < len(c.items)-1 {
				c.selectedIdx++
			}
			return c, nil
		}
	}

	return c, nil
}

func (c *simpleListCmp[T]) BindingKeys() []key.Binding {
	return layout.KeyMapToSlice(simpleListKeys)
}

func (c *simpleListCmp[T]) GetSelectedItem() (T, int) {
	if len(c.items) > 0 {
		return c.items[c.selectedIdx], c.selectedIdx
	}

	var zero T
	return zero, -1
}

func (c *simpleListCmp[T]) SetItems(items []T) {
	c.selectedIdx = 0
	c.items = items
}

func (c *simpleListCmp[T]) View() string {
	t := theme.CurrentTheme()
	baseStyle := styles.BaseStyle()

	items := c.items
	maxWidth := 80
	maxVisibleItems := 7
	startIdx := 0

	if len(items) <= 0 {
		return "No content available"
	}

	if len(items) > maxVisibleItems {
		halfVisible := maxVisibleItems / 2
		if c.selectedIdx >= halfVisible && c.selectedIdx < len(items)-halfVisible {
			startIdx = c.selectedIdx - halfVisible
		} else if c.selectedIdx >= len(items)-halfVisible {
			startIdx = len(items) - maxVisibleItems
		}
	}

	endIdx := min(startIdx+maxVisibleItems, len(items))

	listItems := make([]string, 0, maxVisibleItems)

	for i := startIdx; i < endIdx; i++ {
		item := items[i]
		title := item.Render(i == c.selectedIdx, maxWidth)
		listItems = append(listItems, title)
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		baseStyle.
			Background(t.Background()).
			Width(maxWidth).
			Padding(0, 1).
			Render(
				lipgloss.
					JoinVertical(lipgloss.Left, listItems...),
			),
	)
}

func NewSimpleList[T SimpleListItem](items []T) SimpleList[T] {
	return &simpleListCmp[T]{
		items:       items,
		selectedIdx: 0,
	}
}
