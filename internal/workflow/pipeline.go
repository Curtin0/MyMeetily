package workflow

import (
	"context"
	"errors"

	"github.com/mymeetily/mymeetily/internal/agent"
	"github.com/mymeetily/mymeetily/internal/config"
)

func RunMeetingPipeline(ctx context.Context, cfg *config.Config, audioPath, language string, progressFn func(string)) (*agent.MeetingState, error) {
	graph, err := agent.BuildGraph(cfg)
	if err != nil {
		return nil, err
	}

	state := &agent.MeetingState{
		AudioFilePath: audioPath,
		Language:      language,
		ProgressFn:    progressFn,
	}

	result, err := graph.Invoke(ctx, state)
	if err != nil {
		return nil, err
	}
	if result.Error != "" {
		return nil, errors.New(result.Error)
	}
	return result, nil
}
