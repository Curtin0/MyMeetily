package agent

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	"github.com/mymeetily/mymeetily/internal/asr"
	"github.com/mymeetily/mymeetily/internal/config"
	"github.com/mymeetily/mymeetily/internal/ollama"
	"github.com/mymeetily/mymeetily/internal/output"
)

type Nodes struct {
	cfg        *config.Config
	asrClient  *asr.Client
	ollama     *ollama.Client
}

func NewNodes(cfg *config.Config) *Nodes {
	return &Nodes{
		cfg:       cfg,
		asrClient: asr.NewClient(cfg.ASR.WhisperBinary, cfg.ASR.ModelPath, cfg.ASR.Language),
		ollama:    ollama.NewClient(cfg.Ollama.Endpoint, cfg.Ollama.Model, cfg.Ollama.Temperature, cfg.Ollama.MaxTokens),
	}
}

func (n *Nodes) PreprocessAudio(ctx context.Context, state *MeetingState) (*MeetingState, error) {
	if state.ProgressFn != nil {
		state.ProgressFn("音频预处理")
	}
	slog.Info("preprocessing audio", "file", state.AudioFilePath)

	// Pass audio file directly to ASR — whisper.cpp supports common formats natively.
	state.WavFilePath = state.AudioFilePath
	return state, nil
}

func (n *Nodes) CallASR(ctx context.Context, state *MeetingState) (*MeetingState, error) {
	if state.ProgressFn != nil {
		state.ProgressFn("语音识别")
	}
	slog.Info("calling ASR", "wav", state.WavFilePath)

	result, err := n.asrClient.Transcribe(ctx, state.WavFilePath, state.Language)
	if err != nil {
		state.Error = fmt.Sprintf("ASR failed: %v", err)
		return state, fmt.Errorf("asr: %w", err)
	}

	state.RawTranscript = result.Text
	state.Segments = make([]Segment, len(result.Segments))
	for i, seg := range result.Segments {
		state.Segments[i] = Segment{
			Start:      seg.Start,
			End:        seg.End,
			Text:       seg.Text,
			Confidence: seg.Confidence,
		}
	}
	state.AudioDuration = result.Duration

	slog.Info("ASR done", "text_len", len(state.RawTranscript), "duration", state.AudioDuration)
	return state, nil
}

func (n *Nodes) Summarize(ctx context.Context, state *MeetingState) (*MeetingState, error) {
	if state.ProgressFn != nil {
		state.ProgressFn("LLM 整理")
	}
	slog.Info("summarizing with Ollama", "endpoint", n.cfg.Ollama.Endpoint, "model", n.cfg.Ollama.Model)

	resp, err := n.ollama.Summarize(ctx, state.RawTranscript)
	if err != nil {
		state.Error = fmt.Sprintf("Ollama summarization failed: %v", err)
		return state, fmt.Errorf("summarize: %w", err)
	}

	state.MeetingNotes = resp
	slog.Info("summarization done", "notes_len", len(state.MeetingNotes))
	return state, nil
}

func (n *Nodes) FormatOutput(ctx context.Context, state *MeetingState) (*MeetingState, error) {
	if state.ProgressFn != nil {
		state.ProgressFn("生成输出")
	}
	baseName := strings.TrimSuffix(filepath.Base(state.AudioFilePath), filepath.Ext(state.AudioFilePath))
	timestamp := time.Now().Format("20060102_150405")

	// Output files into the same timestamp directory as the recording.
	outDir := filepath.Dir(state.AudioFilePath)

	fullContent := fmt.Sprintf(`# 会议纪要

> 源文件：%s
> 音频时长：%.0f 秒
> 识别语言：%s
> 生成时间：%s

---

%s
`,
		filepath.Base(state.AudioFilePath),
		state.AudioDuration,
		state.Language,
		time.Now().Format("2006-01-02 15:04:05"),
		state.MeetingNotes,
	)

	// Save meeting notes as styled HTML.
	htmlPath := filepath.Join(outDir, fmt.Sprintf("%s_纪要_%s.html", baseName, timestamp))
	if err := output.WriteHTML(fullContent, htmlPath); err != nil {
		slog.Error("write html", "error", err)
	}

	// Save raw ASR transcript as plain text.
	state.TranscriptPath = filepath.Join(outDir, fmt.Sprintf("%s_转写_%s.txt", baseName, timestamp))
	if err := output.WriteText(state.RawTranscript, state.TranscriptPath); err != nil {
		slog.Error("write transcript", "error", err)
	}

	state.MeetingNotes = fullContent
	state.OutputPath = htmlPath
	state.HTMLOutputPath = htmlPath

	slog.Info("output formatted", "dir", outDir, "html", htmlPath, "txt", state.TranscriptPath)
	return state, nil
}
