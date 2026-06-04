package agent

import (
	"context"
	"fmt"
	"log/slog"
)

func (n *Nodes) Summarize(ctx context.Context, state *MeetingState) (*MeetingState, error) {
	if state.ProgressFn != nil {
		state.ProgressFn("LLM 整理")
	}
	slog.Info("summarizing with Ollama", "endpoint", n.cfg.Ollama.Endpoint, "model", n.cfg.Ollama.Model)

	resp, err := n.ollama.Summarize(ctx, state.RawTranscript)
	if err != nil {
		state.SummaryError = err.Error()
		state.SummaryContent = fallbackMeetingNotes(state.SummaryError)
		state.MeetingNotes = state.SummaryContent
		slog.Warn("summarization failed, falling back to transcript-only output", "error", err)
		return state, nil
	}

	state.SummaryEnabled = true
	state.SummaryContent = resp
	state.MeetingNotes = resp
	slog.Info("summarization done", "notes_len", len(state.MeetingNotes))
	return state, nil
}

func fallbackMeetingNotes(summaryErr string) string {
	return fmt.Sprintf(`## 摘要生成失败

本次未能完成自动总结，已保留完整转写内容。
- 可能原因：Ollama 服务未启动、模型不可用，或本次请求超时
- 错误信息：%s
- 建议：检查 Ollama 状态后重试；当前 HTML 报告下方仍可展开查看完整原文
`, summaryErr)
}
