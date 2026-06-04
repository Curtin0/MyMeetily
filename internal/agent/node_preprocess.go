package agent

import (
	"context"
	"log/slog"
)

func (n *Nodes) PreprocessAudio(ctx context.Context, state *MeetingState) (*MeetingState, error) {
	if state.ProgressFn != nil {
		state.ProgressFn("音频预处理")
	}
	slog.Info("preprocessing audio", "file", state.AudioFilePath)

	state.WavFilePath = state.AudioFilePath
	return state, nil
}
