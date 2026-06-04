package agent

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	"github.com/mymeetily/mymeetily/internal/output"
)

func (n *Nodes) FormatOutput(ctx context.Context, state *MeetingState) (*MeetingState, error) {
	if state.ProgressFn != nil {
		state.ProgressFn("生成输出")
	}
	baseName := strings.TrimSuffix(filepath.Base(state.AudioFilePath), filepath.Ext(state.AudioFilePath))
	timestamp := time.Now().Format("20060102_150405")
	outDir := filepath.Dir(state.AudioFilePath)

	fullContent := BuildMeetingReportMarkdown(state, time.Now())
	transcriptSegments := toOutputSegments(state.Segments)

	htmlPath := filepath.Join(outDir, fmt.Sprintf("%s_纪要_%s.html", baseName, timestamp))
	if err := output.WriteHTML(fullContent, state.RawTranscript, transcriptSegments, htmlPath); err != nil {
		slog.Error("write html", "error", err)
	}

	state.MarkdownPath = filepath.Join(outDir, fmt.Sprintf("%s_纪要_%s.md", baseName, timestamp))
	if err := output.WriteMarkdown(fullContent, state.MarkdownPath); err != nil {
		slog.Error("write markdown", "error", err)
	}

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

func BuildMeetingReportMarkdown(state *MeetingState, generatedAt time.Time) string {
	if state == nil {
		return ""
	}

	return fmt.Sprintf(`# 会议纪要

> 源文件：%s
> 音频时长：%.0f 秒
> 识别语言：%s
> 总结状态：%s
> 生成时间：%s

---

%s
`,
		filepath.Base(state.AudioFilePath),
		state.AudioDuration,
		state.Language,
		summaryStatusText(state),
		generatedAt.Format("2006-01-02 15:04:05"),
		state.SummaryContent,
	)
}

func summaryStatusText(state *MeetingState) string {
	if state == nil {
		return "未启用自动总结，仅输出原文转写"
	}
	if state.SummaryEnabled && state.SummaryError == "" {
		return "已启用自动总结"
	}
	return "未启用自动总结，仅输出原文转写"
}

func toOutputSegments(segments []Segment) []output.TranscriptSegment {
	out := make([]output.TranscriptSegment, 0, len(segments))
	for _, seg := range segments {
		out = append(out, output.TranscriptSegment{
			Start: seg.Start,
			End:   seg.End,
			Text:  seg.Text,
		})
	}
	return out
}
