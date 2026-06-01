package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/mymeetily/mymeetily/internal/audio"
)

// MicSelectModel lets the user choose mic and system audio devices.
type MicSelectModel struct {
	backend        audio.Backend
	micDevices     []string
	speakerDevices []string
	micCursor      int
	speakerCursor  int
	detecting      bool
	ready          bool
	err            error
	step           int // 0 = pick mic, 1 = pick speaker
	micDevice      string
}

func NewMicSelectModel(backend string) MicSelectModel {
	return MicSelectModel{
		backend:   audio.BackendWASAPI,
		detecting: true,
	}
}

type micDevicesMsg struct {
	devices []string
	err     error
}

type speakerDevicesMsg struct {
	devices []string
	err     error
}

func detectMicDevices(backend audio.Backend) tea.Cmd {
	return func() tea.Msg {
		devices, err := audio.ListCaptureDevices(backend)
		return micDevicesMsg{devices: devices, err: err}
	}
}

func detectSpeakerDevices(backend audio.Backend) tea.Cmd {
	return func() tea.Msg {
		devices, err := audio.ListLoopbackDevices(backend)
		return speakerDevicesMsg{devices: devices, err: err}
	}
}

func (m MicSelectModel) Init() tea.Cmd {
	return detectMicDevices(m.backend)
}

func (m *Model) updateMicSelect(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case micDevicesMsg:
		if msg.err != nil {
			m.micselect.err = msg.err
			m.micselect.detecting = false
			return m, nil
		}
		m.micselect.micDevices = msg.devices
		m.micselect.detecting = false
		m.micselect.ready = true
		return m, nil

	case speakerDevicesMsg:
		if msg.err != nil {
			m.micselect.err = msg.err
			return m, nil
		}
		m.micselect.speakerDevices = msg.devices
		m.micselect.speakerCursor = len(msg.devices)
		return m, nil

	case tea.KeyMsg:
		if !m.micselect.ready {
			if msg.String() == "esc" || msg.String() == "b" {
				m.screen = ScreenMenu
				return m, m.menu.Init()
			}
			return m, nil
		}

		switch m.micselect.step {
		case 0: // selecting microphone
			switch msg.String() {
			case "up", "k":
				if m.micselect.micCursor > 0 {
					m.micselect.micCursor--
				}
			case "down", "j":
				if m.micselect.micCursor < len(m.micselect.micDevices)-1 {
					m.micselect.micCursor++
				}
			case "enter", " ":
				if len(m.micselect.micDevices) > 0 {
					m.micselect.micDevice = m.micselect.micDevices[m.micselect.micCursor]
					m.micselect.step = 1
					m.micselect.err = nil
					return m, detectSpeakerDevices(m.micselect.backend)
				}
			}

		case 1: // selecting system audio
			sp := m.micselect.speakerDevices
			switch msg.String() {
			case "up", "k":
				if m.micselect.speakerCursor > 0 {
					m.micselect.speakerCursor--
				}
			case "down", "j":
				if m.micselect.speakerCursor < len(sp) {
					m.micselect.speakerCursor++
				}
			case "enter", " ":
				if m.micselect.speakerCursor < len(sp) {
					return m.goToRecord(m.micselect.micDevice, sp[m.micselect.speakerCursor])
				}
				return m.goToRecord(m.micselect.micDevice, "")
			}
		}

		if msg.String() == "esc" || msg.String() == "b" {
			if m.micselect.step == 1 {
				m.micselect.step = 0
				return m, nil
			}
			m.screen = ScreenMenu
			return m, m.menu.Init()
		}
	}

	return m, nil
}

func (m MicSelectModel) View() string {
	var b strings.Builder

	b.WriteString(RenderTitle())
	b.WriteString("\n")

	if m.detecting {
		b.WriteString(SubtitleStyle.Render("正在检测音频设备..."))
		return lipgloss.NewStyle().Padding(4, 6).Render(b.String())
	}

	if m.err != nil {
		b.WriteString(SubtitleStyle.Render("设备检测失败"))
		b.WriteString("\n\n")
		b.WriteString(lipgloss.NewStyle().Foreground(Error).Render(m.err.Error()))
		b.WriteString("\n\n")
		b.WriteString(RenderHelp("b", "返回"))
		return lipgloss.NewStyle().Padding(4, 6).Render(b.String())
	}

	if m.step == 0 {
		// Step 1: Pick microphone
		b.WriteString(SubtitleStyle.Render("🎤 选择麦克风"))
		b.WriteString("\n\n")
		b.WriteString(lipgloss.NewStyle().Foreground(DimText).Render("请选择录音用的麦克风设备："))
		b.WriteString("\n\n")

		for i, dev := range m.micDevices {
			if i == m.micCursor {
				b.WriteString(ActiveItemStyle.Render("▸ " + dev))
			} else {
				b.WriteString(InactiveItemStyle.Render("  " + dev))
			}
			b.WriteString("\n")
		}

		b.WriteString("\n")
		b.WriteString(RenderHelp(
			"↑/↓", "选择",
			"Enter", "确认",
			"b", "返回",
		))
	} else {
		// Step 2: Pick system audio
		b.WriteString(SubtitleStyle.Render("🔊 选择系统音频"))
		b.WriteString("\n\n")
		b.WriteString(lipgloss.NewStyle().Foreground(DimText).Render(
			"麦克风: " + lipgloss.NewStyle().Foreground(Success).Render(m.micDevice)))
		b.WriteString("\n\n")

		if len(m.speakerDevices) == 0 {
			b.WriteString(lipgloss.NewStyle().Foreground(DimText).Render("正在检测系统音频设备..."))
		} else {
			b.WriteString(lipgloss.NewStyle().Foreground(DimText).Render("选择要做回环采集的系统输出端点："))
			b.WriteString("\n\n")

			for i, dev := range m.speakerDevices {
				prefix := "  "
				if i == m.speakerCursor {
					prefix = "▸ "
				}
				b.WriteString(lipgloss.NewStyle().Render(prefix + dev))
				b.WriteString("\n")
			}

			nonePrefix := "  "
			if m.speakerCursor == len(m.speakerDevices) {
				nonePrefix = "▸ "
			}
			b.WriteString(lipgloss.NewStyle().Foreground(DimText).Render(nonePrefix + "不录制系统声音"))
			b.WriteString("\n\n")

			b.WriteString("\n")
			b.WriteString(lipgloss.NewStyle().Foreground(Warning).Render(
				"💡 提示：WASAPI 会直接从 Windows 播放端点做回环采集，通常不需要立体声混音。"))
		}
	}

	return lipgloss.NewStyle().Padding(4, 6).Render(b.String())
}
