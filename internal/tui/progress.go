package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/mymeetily/mymeetily/internal/config"
)

// Pipeline stages in order.
var pipelineStages = []string{"音频预处理", "语音识别", "LLM 整理", "生成输出"}

// ProgressModel shows a progress bar while the Eino agent graph runs.
type ProgressModel struct {
	spinner      spinner.Model
	audioPath    string
	language     string
	done         bool
	err          error
	currentStage string
	progressCh   chan string
}

func NewProgressModel(sp spinner.Model) ProgressModel {
	return ProgressModel{
		spinner: sp,
	}
}

// Init starts the Eino agent graph (agent runtime), the spinner tick (UI),
// and the progress listener.
func (m ProgressModel) Init(cfg *config.Config, audioPath, language string) tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		runPipeline(cfg, audioPath, language, m.progressCh),
		listenPipelineProgress(m.progressCh),
	)
}

func (m *Model) updateProgress(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.progress.spinner, cmd = m.progress.spinner.Update(msg)
		cmds = append(cmds, cmd)

	case pipelineProgressMsg:
		m.progress.currentStage = msg.stage
		// Re-subscribe for the next progress update
		if m.progress.progressCh != nil {
			cmds = append(cmds, listenPipelineProgress(m.progress.progressCh))
		}

	case pipelineDoneMsg:
		m.progress.done = true
		return m.goToResult(msg.state)

	case pipelineErrMsg:
		m.progress.err = msg.err
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			m.quitting = true
			return m, tea.Quit
		}
		if (msg.String() == "b" || msg.String() == "esc") && (m.progress.done || m.progress.err != nil) {
			m.screen = ScreenMenu
			return m, m.menu.Init()
		}
	}

	// Keep spinner ticking while Eino graph is running
	if !m.progress.done && m.progress.err == nil {
		cmds = append(cmds, m.progress.spinner.Tick)
	}

	return m, tea.Batch(cmds...)
}

func (m ProgressModel) View() string {
	var b strings.Builder

	b.WriteString(RenderTitle())
	b.WriteString("\n")
	b.WriteString(SubtitleStyle.Render("处理中..."))
	b.WriteString("\n\n")

	fileLabel := lipgloss.NewStyle().Foreground(DimText).Render("文件: ")
	filePath := lipgloss.NewStyle().Foreground(BrightText).Render(m.audioPath)
	b.WriteString(fileLabel + filePath + "\n\n")

	if m.err != nil {
		b.WriteString(lipgloss.NewStyle().Foreground(Error).Render("✗ 处理失败: " + m.err.Error()))
		b.WriteString("\n\n")
		b.WriteString(RenderHelp("b", "返回主菜单", "q", "退出"))
	} else if m.done {
		b.WriteString(lipgloss.NewStyle().Foreground(Success).Render("✓ 处理完成，正在加载结果..."))
	} else {
		// Spinner + current stage title
		stageLabel := m.currentStage
		if stageLabel == "" {
			stageLabel = "准备中..."
		}
		spinnerText := m.spinner.View() + " " +
			lipgloss.NewStyle().Foreground(Primary).Render("Eino Agent 正在执行管道...")
		b.WriteString(spinnerText)
		b.WriteString("\n\n")

		// Progress bar
		currentIdx := stageIndex(m.currentStage)
		if currentIdx < 0 {
			currentIdx = -1 // Show nothing filled if no stage yet
		}
		total := len(pipelineStages)
		barWidth := 30

		// Build the progress bar
		var barBuilder strings.Builder
		barBuilder.WriteString("[")
		for i := 0; i < barWidth; i++ {
			threshold := (i + 1) * total / barWidth
			if currentIdx >= 0 && threshold <= currentIdx+1 {
				barBuilder.WriteString("█")
			} else {
				barBuilder.WriteString("░")
			}
		}
		barBuilder.WriteString("]")

		barStyle := lipgloss.NewStyle().Foreground(Primary)
		if currentIdx >= 0 {
			b.WriteString("  " + barStyle.Render(barBuilder.String()) +
				fmt.Sprintf("  %d/%d  ", currentIdx+1, total) +
				lipgloss.NewStyle().Foreground(BrightText).Render(stageLabel+"中..."))
		} else {
			b.WriteString("  " + lipgloss.NewStyle().Foreground(DimText).Render(barBuilder.String()) +
				"  0/" + fmt.Sprintf("%d  ", total) +
				lipgloss.NewStyle().Foreground(DimText).Render(stageLabel))
		}
		b.WriteString("\n\n")

		// Stage step indicators
		var parts []string
		for i, s := range pipelineStages {
			var icon string
			var style lipgloss.Style
			if i < currentIdx {
				icon = "✓"
				style = lipgloss.NewStyle().Foreground(Success)
			} else if i == currentIdx {
				icon = "●"
				style = lipgloss.NewStyle().Foreground(Primary).Bold(true)
			} else {
				icon = "○"
				style = lipgloss.NewStyle().Foreground(DimText)
			}
			parts = append(parts, style.Render(icon+" "+s))
		}
		b.WriteString("  " + strings.Join(parts, "  →  "))
	}

	return lipgloss.NewStyle().Padding(2, 4).Render(b.String())
}

func stageIndex(stage string) int {
	for i, s := range pipelineStages {
		if s == stage {
			return i
		}
	}
	return -1
}
