package agent

import (
	"github.com/mymeetily/mymeetily/internal/asr"
	"github.com/mymeetily/mymeetily/internal/config"
	"github.com/mymeetily/mymeetily/internal/ollama"
)

type Nodes struct {
	cfg       *config.Config
	asrClient *asr.Client
	ollama    *ollama.Client
}

func NewNodes(cfg *config.Config) *Nodes {
	return &Nodes{
		cfg:       cfg,
		asrClient: asr.NewClient(cfg.ASR.WhisperBinary, cfg.ASR.ModelPath, cfg.ASR.Language),
		ollama:    ollama.NewClient(cfg.Ollama.Endpoint, cfg.Ollama.Model, cfg.Ollama.Temperature, cfg.Ollama.MaxTokens),
	}
}
