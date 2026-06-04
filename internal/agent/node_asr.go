package agent

import (
	"context"
	"fmt"
	"log/slog"
)

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
	state.Segments = cleanSegments(state.Segments)
	state.RawTranscript = buildTranscriptFromSegments(state.Segments, state.RawTranscript)
	state.AudioDuration = result.Duration

	slog.Info("ASR done", "text_len", len(state.RawTranscript), "duration", state.AudioDuration)
	return state, nil
}
