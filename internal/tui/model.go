package tui

import (
	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/mymeetily/mymeetily/internal/agent"
	"github.com/mymeetily/mymeetily/internal/audio"
	"github.com/mymeetily/mymeetily/internal/config"
)

type Screen int

const (
	ScreenSplash Screen = iota
	ScreenMenu
	ScreenCheck
	ScreenFilePicker
	ScreenMicSelect
	ScreenRecord
	ScreenProgress
	ScreenResult
)

type Model struct {
	screen         Screen
	width          int
	height         int
	cfg            *config.Config
	summaryEnabled bool
	quitting       bool

	splash     SplashModel
	menu       MenuModel
	check      CheckModel
	filepicker FilePickerModel
	micselect  MicSelectModel
	record     RecordModel
	progress   ProgressModel
	result     ResultModel
}

func New(cfg *config.Config, summaryEnabled bool) Model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(Primary)

	fp := filepicker.New()
	fp.CurrentDirectory = "."
	fp.AllowedTypes = []string{".mp3", ".wav", ".m4a", ".flac", ".ogg", ".wma", ".aac"}
	fp.FileAllowed = true
	fp.DirAllowed = false

	vp := viewport.New(80, 20)
	vp.Style = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(Muted)

	return Model{
		screen:         ScreenSplash,
		cfg:            cfg,
		summaryEnabled: summaryEnabled,
		splash:         NewSplashModel(),
		menu:           NewMenuModel(summaryEnabled),
		check:          NewCheckModel(cfg),
		micselect:      NewMicSelectModel(cfg.Audio.Backend),
		record:         NewRecordModel(cfg.Output.OutputDir, summaryEnabled),
		progress:       NewProgressModel(sp),
		filepicker: FilePickerModel{
			FilePicker: fp,
			selected:   false,
		},
		result: ResultModel{
			Viewport: vp,
		},
	}
}

func (m Model) Init() tea.Cmd {
	return m.splash.Init()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.filepicker.FilePicker.Height = msg.Height - 10
		m.result.Viewport.Width = msg.Width - 8
		m.result.Viewport.Height = msg.Height - 10
		return m, nil
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			m.quitting = true
			return m, tea.Quit
		}
	}

	if key, ok := msg.(tea.KeyMsg); ok && key.String() == "esc" {
		if m.screen != ScreenMenu && m.screen != ScreenMicSelect && m.screen != ScreenRecord && m.screen != ScreenSplash {
			m.screen = ScreenMenu
			return m, m.menu.Init()
		}
	}

	switch m.screen {
	case ScreenSplash:
		return m.updateSplash(msg)
	case ScreenMenu:
		return m.updateMenu(msg)
	case ScreenCheck:
		return m.updateCheck(msg)
	case ScreenFilePicker:
		return m.updateFilePicker(msg)
	case ScreenMicSelect:
		return m.updateMicSelect(msg)
	case ScreenRecord:
		return m.updateRecord(msg)
	case ScreenProgress:
		return m.updateProgress(msg)
	case ScreenResult:
		return m.updateResult(msg)
	}

	return m, nil
}

func (m Model) View() string {
	if m.quitting {
		return "Goodbye!\n"
	}

	var content string
	switch m.screen {
	case ScreenSplash:
		content = m.splash.View()
	case ScreenMenu:
		content = m.menu.View()
	case ScreenCheck:
		content = m.check.View()
	case ScreenFilePicker:
		content = m.filepicker.View()
	case ScreenMicSelect:
		content = m.micselect.View()
	case ScreenRecord:
		content = m.record.View()
	case ScreenProgress:
		content = m.progress.View()
	case ScreenResult:
		content = m.result.View()
	}

	if m.width > 0 && m.height > 0 {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
	}
	return content
}

func (m *Model) goToCheck() (tea.Model, tea.Cmd) {
	m.screen = ScreenCheck
	return m, m.check.Init()
}

func (m *Model) goToFilePicker() (tea.Model, tea.Cmd) {
	m.screen = ScreenFilePicker
	m.filepicker.selected = false
	return m, m.filepicker.Init()
}

func (m *Model) goToMicSelect() (tea.Model, tea.Cmd) {
	m.screen = ScreenMicSelect
	m.micselect = NewMicSelectModel(m.cfg.Audio.Backend)
	return m, m.micselect.Init()
}

func (m *Model) goToRecord(micDevice, speakerDevice string) (tea.Model, tea.Cmd) {
	m.screen = ScreenRecord
	m.record = NewRecordModel(m.cfg.Output.OutputDir, m.summaryEnabled)
	m.record.transcriber = audio.NewLiveTranscriber(audio.LiveTranscribeConfig{
		WhisperBinary: m.cfg.ASR.WhisperBinary,
		ModelPath:     m.cfg.ASR.ModelPath,
		Language:      m.cfg.ASR.Language,
	})
	return m, tea.Batch(m.record.Init(), StartRecording(m.cfg, micDevice, speakerDevice, m.record.transcriber))
}

func (m *Model) goToProgress(audioPath, language string) (tea.Model, tea.Cmd) {
	m.screen = ScreenProgress
	m.progress = NewProgressModel(m.progress.spinner)
	m.progress.audioPath = audioPath
	m.progress.language = language
	m.progress.progressCh = make(chan string, 8)
	return m, m.progress.Init(m.cfg, audioPath, language)
}

func (m *Model) goToProgressError(audioPath string, err error) (tea.Model, tea.Cmd) {
	m.screen = ScreenProgress
	m.progress = NewProgressModel(m.progress.spinner)
	m.progress.audioPath = audioPath
	m.progress.err = err
	return m, nil
}

func (m *Model) goToResult(state *agent.MeetingState) (tea.Model, tea.Cmd) {
	m.screen = ScreenResult
	m.result.state = state
	return m, m.result.Init()
}
