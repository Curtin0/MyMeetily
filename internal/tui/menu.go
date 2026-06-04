package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type MenuModel struct {
	options        []string
	cursor         int
	selected       bool
	summaryEnabled bool
}

func NewMenuModel(summaryEnabled bool) MenuModel {
	return MenuModel{
		summaryEnabled: summaryEnabled,
		options: []string{
			"实时录音",
			"退出",
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
		prefix := "  "
		if i == m.cursor {
			prefix = "▶ "
			b.WriteString(ActiveItemStyle.Render(prefix + opt))
		} else {
			b.WriteString(InactiveItemStyle.Render(prefix + opt))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	if m.summaryEnabled {
		b.WriteString(lipgloss.NewStyle().Foreground(Success).Render("当前模式：自动总结已启用"))
	} else {
		b.WriteString(lipgloss.NewStyle().Foreground(Warning).Render("当前模式：仅转写，不生成自动总结"))
	}
	b.WriteString("\n\n")
	b.WriteString(RenderHelp(
		"↑/↓", "移动",
		"Enter", "选择",
		"q", "退出",
	))

	return lipgloss.NewStyle().Padding(2, 4).Render(b.String())
}
