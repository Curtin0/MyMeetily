package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SplashModel shows a welcome screen on startup.
type SplashModel struct{}

func NewSplashModel() SplashModel {
	return SplashModel{}
}

func (m SplashModel) Init() tea.Cmd {
	return nil
}

func (m *Model) updateSplash(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := msg.(tea.KeyMsg); ok {
		m.screen = ScreenMenu
		return m, m.menu.Init()
	}
	return m, nil
}

func (m SplashModel) View() string {
	// Box border style
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Primary).
		Padding(2, 4).
		Width(66)

	// Title
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(Primary).
		Align(lipgloss.Center).
		Width(56).
		Render("MyMeetily")

	subtitle := lipgloss.NewStyle().
		Foreground(DimText).
		Align(lipgloss.Center).
		Width(56).
		Render("本地离线 AI 会议助手")

	// Info section — two fixed-width columns for alignment.
	labelStyle := lipgloss.NewStyle().Foreground(DimText).Width(12).Align(lipgloss.Left)
	valStyle := lipgloss.NewStyle().Foreground(BrightText).Width(30).Align(lipgloss.Left)

	info := strings.Builder{}
	addLine := func(label, value string) {
		info.WriteString("  ")
		info.WriteString(labelStyle.Render(label))
		info.WriteString(valStyle.Render(value))
		info.WriteString("\n")
	}
	addLine("语音模型", "Whisper Large-v3 Turbo Q5_0")
	addLine("引擎", "whisper.cpp + Ollama")
	addLine("整理模型", "Ollama 本机服务")
	addLine("输出格式", "HTML + TXT")

	// Mode
	modes := lipgloss.NewStyle().
		Foreground(DimText).
		Width(56).
		Align(lipgloss.Center).
		Render("🎤 实时录音转录")

	// Press key hint
	hint := lipgloss.NewStyle().
		Foreground(Muted).
		Blink(true).
		Align(lipgloss.Center).
		Width(56).
		Render("按任意键进入主菜单 . . .")

	// Assemble box
	content := lipgloss.JoinVertical(lipgloss.Center,
		"",
		title,
		subtitle,
		"",
		info.String(),
		modes,
		"",
		hint,
	)

	return lipgloss.Place(0, 0, lipgloss.Center, lipgloss.Center,
		boxStyle.Render(content),
	)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
