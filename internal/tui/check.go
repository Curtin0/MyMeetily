package tui

import (
	"context"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/mymeetily/mymeetily/internal/config"
	"github.com/mymeetily/mymeetily/internal/ollama"
)

type checkStatus int

const (
	checkPending checkStatus = iota
	checkRunning
	checkOK
	checkFail
)

type serviceCheck struct {
	name   string
	status checkStatus
	err    error
}

// CheckModel displays dependency status.
type CheckModel struct {
	checks  []serviceCheck
	done    bool
	running bool
}

func NewCheckModel(cfg *config.Config) CheckModel {
	return CheckModel{
		checks: []serviceCheck{
			{name: "whisper.cpp (" + cfg.ASR.WhisperBinary + ")", status: checkPending},
			{name: "模型文件 (" + cfg.ASR.ModelPath + ")", status: checkPending},
			{name: "Ollama 服务 (" + cfg.Ollama.Endpoint + ")", status: checkPending},
		},
	}
}

type checkResultMsg struct {
	index int
	ok    bool
	err   error
}

func (m CheckModel) Init() tea.Cmd {
	return runChecks
}

func runChecks() tea.Msg {
	return checkAllMsg{}
}

type checkAllMsg struct{}

func (m *Model) updateCheck(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case checkAllMsg:
		m.check.running = true
		return m, tea.Batch(
			checkWhisperBinaryCmd(0, m.cfg.ASR.WhisperBinary),
			checkModelFileCmd(1, m.cfg.ASR.ModelPath),
			checkOllamaServerCmd(2, m.cfg.Ollama.Endpoint),
		)

	case checkResultMsg:
		if msg.ok {
			m.check.checks[msg.index].status = checkOK
		} else {
			m.check.checks[msg.index].status = checkFail
			m.check.checks[msg.index].err = msg.err
		}

		allDone := true
		for _, c := range m.check.checks {
			if c.status == checkPending || c.status == checkRunning {
				allDone = false
				break
			}
		}
		if allDone {
			m.check.done = true
			m.check.running = false
		}

		return m, nil

	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "b" {
			m.screen = ScreenMenu
			m.check = NewCheckModel(m.cfg)
			return m, m.menu.Init()
		}
	}

	return m, nil
}

func checkWhisperBinaryCmd(index int, binary string) tea.Cmd {
	return func() tea.Msg {
		_, err := exec.LookPath(binary)
		return checkResultMsg{index: index, ok: err == nil, err: err}
	}
}

func checkModelFileCmd(index int, path string) tea.Cmd {
	return func() tea.Msg {
		_, err := os.Stat(path)
		return checkResultMsg{index: index, ok: err == nil, err: err}
	}
}

func checkOllamaServerCmd(index int, endpoint string) tea.Cmd {
	return func() tea.Msg {
		client := ollama.NewClient(endpoint, "qwen2.5:1.5b-instruct", 0.3, 4096)
		err := client.Ping(context.Background())
		return checkResultMsg{index: index, ok: err == nil, err: err}
	}
}

func (m CheckModel) View() string {
	var b strings.Builder

	b.WriteString(RenderTitle())
	b.WriteString("\n")
	b.WriteString(SubtitleStyle.Render("依赖检查"))
	b.WriteString("\n\n")

	for _, c := range m.checks {
		var icon string
		switch c.status {
		case checkOK:
			icon = SuccessStyle.Render()
		case checkFail:
			icon = ErrorStyle.Render()
		case checkRunning:
			icon = PendingStyle.Render()
		default:
			icon = "  "
		}

		statusText := c.name
		if c.status == checkFail && c.err != nil {
			statusText += " — " + lipgloss.NewStyle().Foreground(Error).Render(c.err.Error())
		} else if c.status == checkOK {
			statusText += " — " + lipgloss.NewStyle().Foreground(Success).Render("可用")
		} else if c.status == checkPending || c.status == checkRunning {
			statusText += " — 检测中..."
		}

		b.WriteString(icon + "  " + statusText + "\n")
	}

	b.WriteString("\n")
	if m.done {
		allOK := true
		for _, c := range m.checks {
			if c.status != checkOK {
				allOK = false
				break
			}
		}
		if allOK {
			b.WriteString(lipgloss.NewStyle().Foreground(Success).Render("所有依赖已就绪！"))
		} else {
			b.WriteString(lipgloss.NewStyle().Foreground(Error).Render("部分依赖未就绪，请检查后重试。"))
		}
		b.WriteString("\n\n")
		b.WriteString(RenderHelp("b", "返回主菜单", "q", "退出"))
	} else {
		b.WriteString(lipgloss.NewStyle().Foreground(DimText).Render("正在检测..."))
	}

	return lipgloss.NewStyle().Padding(2, 4).Render(b.String())
}
