package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/mymeetily/mymeetily/internal/agent"
)

// ResultModel displays the final meeting notes.
type ResultModel struct {
	Viewport viewport.Model
	state    *agent.MeetingState
}

func (m ResultModel) Init() tea.Cmd {
	return nil
}

func (m *Model) updateResult(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "b", "esc":
			m.screen = ScreenMenu
			return m, m.menu.Init()
		}
	}

	m.result.Viewport, cmd = m.result.Viewport.Update(msg)
	return m, cmd
}

func (m *ResultModel) View() string {
	if m.state == nil {
		return "没有结果数据。"
	}

	var b strings.Builder

	b.WriteString(RenderTitle())
	b.WriteString("\n")

	stats := fmt.Sprintf("音频时长: %.0f秒  |  语言: %s  |  识别文本: %d字",
		m.state.AudioDuration,
		m.state.Language,
		len(m.state.RawTranscript),
	)
	b.WriteString(lipgloss.NewStyle().Foreground(DimText).Render(stats))
	b.WriteString("\n\n")

	m.Viewport.SetContent(m.state.MeetingNotes)
	b.WriteString(m.Viewport.View())
	b.WriteString("\n")

	// Show generated output files
	b.WriteString(lipgloss.NewStyle().Foreground(DimText).Render("──────────────────────────────"))
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Foreground(Success).Render("✓ 已生成以下文件:"))
	b.WriteString("\n")

	if m.state.HTMLOutputPath != "" {
		b.WriteString("  📄 " + lipgloss.NewStyle().Foreground(BrightText).Render(m.state.HTMLOutputPath))
		b.WriteString("\n")
	}
	if m.state.TranscriptPath != "" {
		b.WriteString("  📝 " + lipgloss.NewStyle().Foreground(BrightText).Render(m.state.TranscriptPath))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(RenderHelp(
		"↑/↓/PgUp/PgDn", "滚动",
		"b/esc", "返回主菜单",
		"q", "退出",
	))

	return lipgloss.NewStyle().Padding(2, 4).Render(b.String())
}
