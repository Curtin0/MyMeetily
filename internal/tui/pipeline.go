package tui

import (
	"context"
	"errors"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/mymeetily/mymeetily/internal/agent"
	"github.com/mymeetily/mymeetily/internal/config"
)

// pipelineDoneMsg signals that the Eino agent graph has finished.
type pipelineDoneMsg struct {
	state *agent.MeetingState
}

// pipelineErrMsg signals that the Eino agent graph failed.
type pipelineErrMsg struct {
	err error
}

// pipelineProgressMsg reports the current pipeline stage.
type pipelineProgressMsg struct {
	stage string
}

// runPipeline builds the Eino agent graph and invokes it.
// Progress updates are sent through progressCh (non-blocking, drops if full).
func runPipeline(cfg *config.Config, audioPath, language string, progressCh chan<- string) tea.Cmd {
	return func() tea.Msg {
		defer close(progressCh)

		graph, err := agent.BuildGraph(cfg)
		if err != nil {
			return pipelineErrMsg{err}
		}

		state := &agent.MeetingState{
			AudioFilePath: audioPath,
			Language:      language,
			ProgressFn: func(stage string) {
				select {
				case progressCh <- stage:
				default:
					// Non-blocking: drop if channel full (prevents deadlock)
				}
			},
		}

		result, err := graph.Invoke(context.Background(), state)
		if err != nil {
			return pipelineErrMsg{err}
		}
		if result.Error != "" {
			return pipelineErrMsg{err: errors.New(result.Error)}
		}

		return pipelineDoneMsg{state: result}
	}
}

// listenPipelineProgress returns a command that waits for a progress update
// from the channel. Returns nil when the channel is closed.
func listenPipelineProgress(progressCh <-chan string) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-progressCh
		if !ok {
			return nil
		}
		return pipelineProgressMsg{stage: msg}
	}
}
