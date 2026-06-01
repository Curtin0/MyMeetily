package tui

import (
	"fmt"
	"math"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/mymeetily/mymeetily/internal/audio"
	"github.com/mymeetily/mymeetily/internal/config"
)

// RecordModel is the live recording TUI screen.
type RecordModel struct {
	mic        audio.Recorder
	outputDir  string
	elapsed    time.Duration
	running    bool
	stopping   bool
	outputPath string
	peakLevel  float32 // current mic volume (0.0–1.0)

	// Real-time subtitles.
	transcriber *audio.LiveTranscriber
	transcript  []string // accumulated transcribed sentences (newest last)
}

func NewRecordModel(outputDir string) RecordModel {
	return RecordModel{
		outputDir: outputDir,
	}
}

type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m RecordModel) Init() tea.Cmd {
	return nil
}

type recordStartedMsg struct {
	mic audio.Recorder
}

type recordStoppedMsg struct {
	outputPath string
}

type recordFailedMsg struct {
	outputPath string
	err        error
}

// StartRecording begins WASAPI capture (mic + optional system audio loopback).
// lt is the LiveTranscriber for real-time subtitles; may be nil.
func StartRecording(cfg *config.Config, micDevice, speakerDevice string, lt *audio.LiveTranscriber) tea.Cmd {
	return func() tea.Msg {
		outputPath := audio.BuildRecordingPath(cfg.Output.OutputDir, time.Now())

		var onAudio func([]byte, audio.AudioFormat)
		if lt != nil {
			onAudio = lt.Feed
		}

		recorder, err := audio.NewRecorder(audio.RecorderConfig{
			MicDevice:     micDevice,
			SpeakerDevice: speakerDevice,
			OutputPath:    outputPath,
			OnAudioData:   onAudio,
		})
		if err != nil {
			return recordFailedMsg{outputPath: outputPath, err: fmt.Errorf("无法创建录音器: %w", err)}
		}

		if err := recorder.Start(); err != nil {
			return recordFailedMsg{outputPath: outputPath, err: fmt.Errorf("无法启动录音: %w", err)}
		}

		return recordStartedMsg{mic: recorder}
	}
}

func stopRecording(mic audio.Recorder) tea.Cmd {
	return func() tea.Msg {
		if err := mic.Stop(); err != nil {
			return recordFailedMsg{outputPath: mic.OutputPath(), err: fmt.Errorf("录音保存失败: %w", err)}
		}
		return recordStoppedMsg{outputPath: mic.OutputPath()}
	}
}

func (m *Model) updateRecord(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case recordStartedMsg:
		m.record.mic = msg.mic
		m.record.running = true
		return m, tickCmd()

	case tickMsg:
		if m.record.running && m.record.mic != nil {
			m.record.elapsed = m.record.mic.Elapsed()
			m.record.peakLevel = m.record.mic.PeakLevel()
		}
		// Drain any new transcription results.
		if m.record.transcriber != nil {
			m.record.drainTranscript()
		}
		if m.record.running {
			return m, tickCmd()
		}
		return m, nil

	case recordStoppedMsg:
		m.record.running = false
		m.record.stopping = false
		m.record.outputPath = msg.outputPath
		// Drain any remaining transcript.
		if m.record.transcriber != nil {
			m.record.transcriber.Close()
			m.record.drainTranscript()
			m.record.transcriber = nil
		}
		return m.goToProgress(msg.outputPath, m.cfg.ASR.Language)

	case recordFailedMsg:
		m.record.running = false
		m.record.stopping = false
		m.record.outputPath = msg.outputPath
		if m.record.transcriber != nil {
			m.record.transcriber.Close()
			m.record.transcriber = nil
		}
		return m.goToProgressError(msg.outputPath, msg.err)

	case tea.KeyMsg:
		switch msg.String() {
		case "enter", " ":
			if m.record.running {
				m.record.stopping = true
				return m, stopRecording(m.record.mic)
			}
		case "esc":
			if m.record.running && m.record.mic != nil {
				m.record.mic.Stop()
			}
			if m.record.transcriber != nil {
				m.record.transcriber.Close()
				m.record.transcriber = nil
			}
			m.screen = ScreenMenu
			return m, m.menu.Init()
		}
	}

	return m, nil
}

// drainTranscript reads all pending results from the transcriber without blocking.
func (m *RecordModel) drainTranscript() {
	for {
		select {
		case text, ok := <-m.transcriber.Results:
			if !ok {
				return
			}
			if text == "__pending__" || text == "" {
				continue
			}
			m.transcript = append(m.transcript, text)
			// Keep only the last 20 sentences to limit memory.
			if len(m.transcript) > 20 {
				m.transcript = m.transcript[len(m.transcript)-20:]
			}
		default:
			return
		}
	}
}

func (m RecordModel) View() string {
	var b strings.Builder

	b.WriteString(RenderTitle())
	b.WriteString("\n")
	b.WriteString(SubtitleStyle.Render("🎤 实时录音"))
	b.WriteString("\n\n")

	timerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(BrightText).
		MarginBottom(1)

	elapsed := m.elapsed
	if !m.running && elapsed == 0 {
		elapsed = 0
	}

	h := int(elapsed.Hours())
	min := int(elapsed.Minutes()) % 60
	sec := int(elapsed.Seconds()) % 60
	timerText := fmt.Sprintf("⏱  %02d:%02d:%02d", h, min, sec)
	b.WriteString(timerStyle.Render(timerText))
	b.WriteString("\n")

	if m.stopping {
		b.WriteString(lipgloss.NewStyle().Foreground(Warning).Render("⏳ 正在保存录音..."))
	} else if m.running {
		dot := lipgloss.NewStyle().Foreground(Error).Blink(true).Render("●")
		b.WriteString(dot + " " + lipgloss.NewStyle().Foreground(Error).Render("录制中"))
		b.WriteString("\n\n")
		bar := generateAudioBar(m.peakLevel)
		b.WriteString(lipgloss.NewStyle().Foreground(Primary).Render(bar))
		b.WriteString("\n")
		pct := int(m.peakLevel * 100)
		b.WriteString(lipgloss.NewStyle().Foreground(DimText).Render(
			fmt.Sprintf("音量: %d%%", pct),
		))
	} else {
		b.WriteString(lipgloss.NewStyle().Foreground(DimText).Render("准备开始录音..."))
	}

	b.WriteString("\n\n")

	// Live transcription display.
	if m.running && m.transcriber != nil {
		b.WriteString(lipgloss.NewStyle().Foreground(DimText).Render("──────────────────────────────"))
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Foreground(Secondary).Bold(true).Render("📝 实时字幕"))
		b.WriteString("\n")

		if len(m.transcript) > 0 {
			start := 0
			if len(m.transcript) > 8 {
				start = len(m.transcript) - 8
			}
			for _, line := range m.transcript[start:] {
				b.WriteString("  ")
				b.WriteString(lipgloss.NewStyle().Foreground(BrightText).Render(line))
				b.WriteString("\n")
			}
			b.WriteString("  ")
			b.WriteString(lipgloss.NewStyle().Foreground(DimText).Blink(true).Render("…"))
		} else {
			// Show VAD status for debugging.
			d := m.transcriber.Diag.Snapshot()
			rmsBar := generateMiniBar(d.LastRMS)
			b.WriteString(fmt.Sprintf("  %s 音频:%d 语音:%d 静音:%d | 句子:%d whisper:运行%d 成功%d 空%d 失败%d",
				rmsBar, d.AudioPackets, d.VoiceFrames, d.SilentFrames,
				d.SentencesCut, d.WhisperRunning, d.WhisperOK, d.WhisperEmpty, d.WhisperErr))
			if d.LastResult != "" {
				b.WriteString("\n  上次结果: ")
				b.WriteString(lipgloss.NewStyle().Foreground(BrightText).Render(d.LastResult))
			}
		}
		b.WriteString("\n")
	}

	if m.running {
		b.WriteString(RenderHelp(
			"Enter/Space", "结束录音并生成纪要",
			"Esc", "放弃录音",
		))
	} else if !m.stopping {
		b.WriteString(RenderHelp(
			"Enter", "开始录音",
			"Esc", "返回",
		))
	}

	return lipgloss.NewStyle().Padding(4, 6).Render(b.String())
}

// generateAudioBar renders a 20-column audio level meter from the mic peak level.
// generateMiniBar returns a compact RMS indicator for diagnostics.
func generateMiniBar(rms float32) string {
	bar := []rune(" ▁▂▃▄▅▆█")
	level := int(rms * 7 * 20) // scale up for small RMS values
	if level < 0 {
		level = 0
	}
	if level > 7 {
		level = 7
	}
	return fmt.Sprintf("[%c]", bar[level])
}

func generateAudioBar(peak float32) string {
	bar := []rune("▁▂▃▄▅▆▇█")

	// Use a logarithmic (dB-like) scale so quiet speech is visible.
	// Map -48dB..0dB to bar level 0..7.
	var level int
	if peak < 0.0001 {
		level = 0
	} else {
		db := 20.0 * math.Log10(float64(peak)) // e.g., -20dB for 0.1 peak
		// Clamp to [-48, 0] dB, then scale to 0..7.
		scaled := (db + 48.0) / 48.0 * 7.0
		level = int(scaled)
		if level < 0 {
			level = 0
		}
		if level > 7 {
			level = 7
		}
	}

	var b strings.Builder
	for i := 0; i < 20; i++ {
		// Natural variation around the current level.
		spread := level / 3
		if spread < 1 && level > 0 {
			spread = 1
		}
		offset := (i*7 + i*i%11) % (spread*2 + 1) - spread
		idx := level + offset
		if idx < 0 {
			idx = 0
		}
		if idx > 7 {
			idx = 7
		}
		b.WriteRune(bar[idx])
	}
	return b.String()
}
