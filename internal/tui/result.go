package tui

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/mymeetily/mymeetily/internal/agent"
	"github.com/mymeetily/mymeetily/internal/config"
	"github.com/mymeetily/mymeetily/internal/workflow"
)

type ResultModel struct {
	Viewport     viewport.Model
	state        *agent.MeetingState
	actionStatus string
	actionError  bool
}

type resultActionMsg struct {
	status string
	err    error
	state  *agent.MeetingState
}

func (m ResultModel) Init() tea.Cmd {
	return nil
}

func (m *Model) updateResult(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case resultActionMsg:
		if msg.state != nil {
			m.result.state = msg.state
		}
		m.result.actionStatus = msg.status
		m.result.actionError = msg.err != nil
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "b", "esc":
			m.screen = ScreenMenu
			return m, m.menu.Init()
		case "y":
			if m.result.state != nil {
				return m, copyToClipboardCmd(m.result.state.SummaryContent, "已复制摘要")
			}
		case "t":
			if m.result.state != nil {
				return m, copyToClipboardCmd(m.result.state.RawTranscript, "已复制原文转写")
			}
		case "o":
			if m.result.state != nil {
				return m, openOutputFolderCmd(m.result.state.OutputPath)
			}
		case "r":
			if m.result.state != nil && (!m.result.state.SummaryEnabled || m.result.state.SummaryError != "") {
				return m, retrySummaryCmd(m.cfg, m.result.state)
			}
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

	if m.state.SummaryError != "" {
		b.WriteString(lipgloss.NewStyle().Foreground(Warning).Render("摘要生成失败，当前展示的是兜底说明；完整原文已写入 HTML、Markdown 和 TXT。"))
		b.WriteString("\n\n")
	}

	if m.actionStatus != "" {
		style := lipgloss.NewStyle().Foreground(Success)
		if m.actionError {
			style = lipgloss.NewStyle().Foreground(Error)
		}
		b.WriteString(style.Render(m.actionStatus))
		b.WriteString("\n\n")
	}

	m.Viewport.SetContent(m.state.MeetingNotes)
	b.WriteString(m.Viewport.View())
	b.WriteString("\n")

	b.WriteString(lipgloss.NewStyle().Foreground(DimText).Render("──────────────────────────────"))
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Foreground(Success).Render("已生成以下文件"))
	b.WriteString("\n")

	if m.state.HTMLOutputPath != "" {
		b.WriteString("  HTML  " + lipgloss.NewStyle().Foreground(BrightText).Render(m.state.HTMLOutputPath))
		b.WriteString("\n")
	}
	if m.state.MarkdownPath != "" {
		b.WriteString("  MD    " + lipgloss.NewStyle().Foreground(BrightText).Render(m.state.MarkdownPath))
		b.WriteString("\n")
	}
	if m.state.TranscriptPath != "" {
		b.WriteString("  TXT   " + lipgloss.NewStyle().Foreground(BrightText).Render(m.state.TranscriptPath))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(RenderHelp(
		"↑/↓ PgUp/PgDn", "滚动",
		"y", "复制摘要",
		"t", "复制转写",
		"o", "打开目录",
		"r", "重试总结",
		"b/esc", "返回主菜单",
		"q", "退出",
	))

	return lipgloss.NewStyle().Padding(2, 4).Render(b.String())
}

func copyToClipboardCmd(text, success string) tea.Cmd {
	return func() tea.Msg {
		text = strings.TrimSpace(text)
		if text == "" {
			return resultActionMsg{status: "没有可复制的内容", err: fmt.Errorf("empty clipboard content")}
		}

		cmd := exec.Command("cmd", "/c", "clip")
		cmd.Stdin = strings.NewReader(text)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			msg := "复制失败"
			if stderr.Len() > 0 {
				msg = strings.TrimSpace(stderr.String())
			}
			return resultActionMsg{status: msg, err: err}
		}

		return resultActionMsg{status: success}
	}
}

func openOutputFolderCmd(outputPath string) tea.Cmd {
	return func() tea.Msg {
		dir := filepath.Dir(strings.TrimSpace(outputPath))
		if dir == "." || dir == "" {
			return resultActionMsg{status: "无法确定输出目录", err: fmt.Errorf("invalid output path")}
		}

		cmd := exec.Command("explorer", dir)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			msg := "打开目录失败"
			if stderr.Len() > 0 {
				msg = strings.TrimSpace(stderr.String())
			}
			return resultActionMsg{status: msg, err: err}
		}

		return resultActionMsg{status: "已打开输出目录"}
	}
}

func retrySummaryCmd(cfg *config.Config, state *agent.MeetingState) tea.Cmd {
	return func() tea.Msg {
		updated, err := workflow.RetrySummary(context.Background(), cfg, state)
		if err != nil {
			return resultActionMsg{status: "重试总结失败: " + err.Error(), err: err}
		}
		return resultActionMsg{
			status: "已重新生成总结与报告",
			state:  updated,
		}
	}
}
