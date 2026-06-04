package workflow

import (
	"context"

	"github.com/mymeetily/mymeetily/internal/config"
	"github.com/mymeetily/mymeetily/internal/ollama"
)

type StartupStatus struct {
	SummaryEnabled bool
	SummaryMessage string
}

func DetectSummaryAvailability(ctx context.Context, cfg *config.Config, summaryModelPath, summaryModelURL string) (StartupStatus, error) {
	client := ollama.NewClient(cfg.Ollama.Endpoint, cfg.Ollama.Model, cfg.Ollama.Temperature, cfg.Ollama.MaxTokens)
	if err := client.Ping(ctx); err != nil {
		return StartupStatus{
			SummaryEnabled: false,
			SummaryMessage: "未检测到本机 Ollama 正在运行，本次将继续提供录音与转写，并直接输出报告。",
		}, nil
	}

	if err := client.EnsureModelAvailable(ctx, summaryModelPath, summaryModelURL, false); err != nil {
		return StartupStatus{}, err
	}

	return StartupStatus{
		SummaryEnabled: true,
		SummaryMessage: "Ollama 已就绪，将启用自动总结。",
	}, nil
}
