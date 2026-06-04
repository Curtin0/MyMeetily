package agent

import (
	"context"
	"fmt"

	"github.com/mymeetily/mymeetily/internal/config"
)

func RetrySummary(ctx context.Context, cfg *config.Config, state *MeetingState) (*MeetingState, error) {
	if state == nil {
		return nil, fmt.Errorf("missing meeting state")
	}
	if state.RawTranscript == "" {
		return nil, fmt.Errorf("missing transcript content")
	}

	cloned := *state
	nodes := NewNodes(cfg)

	resp, err := nodes.ollama.Summarize(ctx, cloned.RawTranscript)
	if err != nil {
		return nil, fmt.Errorf("summarize: %w", err)
	}

	cloned.SummaryEnabled = true
	cloned.SummaryError = ""
	cloned.SummaryContent = resp
	cloned.MeetingNotes = resp

	result, err := nodes.FormatOutput(ctx, &cloned)
	if err != nil {
		return nil, fmt.Errorf("format output: %w", err)
	}
	return result, nil
}
