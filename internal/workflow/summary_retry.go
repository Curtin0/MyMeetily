package workflow

import (
	"context"

	"github.com/mymeetily/mymeetily/internal/agent"
	"github.com/mymeetily/mymeetily/internal/config"
)

func RetrySummary(ctx context.Context, cfg *config.Config, state *agent.MeetingState) (*agent.MeetingState, error) {
	return agent.RetrySummary(ctx, cfg, state)
}
