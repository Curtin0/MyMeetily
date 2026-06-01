package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// MenuModel is the main menu screen.
type MenuModel struct {
	options  []string
	cursor   int
	selected bool
}

func NewMenuModel() MenuModel {
	return MenuModel{
		options: []string{
			"🎤  实时录音",
			"🚪  退出",
		},
		cursor: 0,
	}
}

func (m MenuModel) Init() tea.Cmd {
	return nil
}

func (m *Model) updateMenu(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q":
			m.quitting = true
			return m, tea.Quit
		case "up", "k":
			if m.menu.cursor > 0 {
				m.menu.cursor--
			}
		case "down", "j":
			if m.menu.cursor < len(m.menu.options)-1 {
				m.menu.cursor++
			}
		case "enter", " ":
			m.menu.selected = true
			switch m.menu.cursor {
			case 0:
				return m.goToMicSelect()
			case 1:
				m.quitting = true
				return m, tea.Quit
			}
		}
	}
	return m, nil
}

func (m MenuModel) View() string {
	var b strings.Builder

	b.WriteString(RenderTitle())
	b.WriteString("\n\n")

	for i, opt := range m.options {
		if i == m.cursor {
			b.WriteString(ActiveItemStyle.Render("▸ " + opt))
		} else {
			b.WriteString(InactiveItemStyle.Render("  " + opt))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(RenderHelp(
		"↑/↓", "移动",
		"Enter", "选择",
		"q", "退出",
	))

	return lipgloss.NewStyle().Padding(2, 4).Render(b.String())
}
