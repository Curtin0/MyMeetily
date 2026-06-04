package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/mymeetily/mymeetily/internal/agent"
	"github.com/mymeetily/mymeetily/internal/config"
	"github.com/mymeetily/mymeetily/internal/workflow"
)

type pipelineDoneMsg struct {
	state *agent.MeetingState
}

type pipelineErrMsg struct {
	err error
}

type pipelineProgressMsg struct {
	stage string
}

func runPipeline(cfg *config.Config, audioPath, language string, progressCh chan<- string) tea.Cmd {
	return func() tea.Msg {
		defer close(progressCh)

		result, err := workflow.RunMeetingPipeline(context.Background(), cfg, audioPath, language, func(stage string) {
			select {
			case progressCh <- stage:
			default:
			}
		})
		if err != nil {
			return pipelineErrMsg{err: err}
		}

		return pipelineDoneMsg{state: result}
	}
}

func listenPipelineProgress(progressCh <-chan string) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-progressCh
		if !ok {
			return nil
		}
		return pipelineProgressMsg{stage: msg}
	}
}
