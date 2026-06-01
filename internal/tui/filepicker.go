package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/lipgloss"
)

// FilePickerModel wraps the bubbles filepicker for audio file selection.
type FilePickerModel struct {
	FilePicker filepicker.Model
	selected   bool
	err        error
}

func (m FilePickerModel) Init() tea.Cmd {
	return m.FilePicker.Init()
}

type fileSelectedMsg struct {
	path string
}

func (m *Model) updateFilePicker(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	// Handle filepicker internal messages first
	m.filepicker.FilePicker, cmd = m.filepicker.FilePicker.Update(msg)

	// Check if a file was selected
	if didSelect, path := m.filepicker.FilePicker.DidSelectFile(msg); didSelect {
		m.filepicker.selected = true
		// Use the configured default language; user can override later
		return m.goToProgress(path, m.cfg.ASR.Language)
	}

	if didSelect, _ := m.filepicker.FilePicker.DidSelectDisabledFile(msg); didSelect {
		m.filepicker.err = fmt.Errorf("不支持的文件格式，请选择音频文件 (mp3/wav/m4a/flac 等)")
	}

	// Handle key events that aren't consumed by filepicker
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "b":
			m.screen = ScreenMenu
			m.filepicker.selected = false
			return m, m.menu.Init()
		}
	}

	return m, cmd
}

func (m FilePickerModel) View() string {
	var b strings.Builder

	b.WriteString(RenderTitle())
	b.WriteString("\n")
	b.WriteString(SubtitleStyle.Render("选择音频文件"))
	b.WriteString("\n\n")
	b.WriteString(m.FilePicker.View())

	if m.err != nil {
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Foreground(Error).Render(m.err.Error()))
	}

	b.WriteString("\n")
	b.WriteString(RenderHelp(
		"↑/↓", "浏览",
		"Enter", "选择",
		"b", "返回",
		"q", "退出",
	))

	return lipgloss.NewStyle().Padding(2, 4).Render(b.String())
}
