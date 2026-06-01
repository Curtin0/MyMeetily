package agent

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/compose"

	"github.com/mymeetily/mymeetily/internal/config"
)

func BuildGraph(cfg *config.Config) (compose.Runnable[*MeetingState, *MeetingState], error) {
	g := compose.NewGraph[*MeetingState, *MeetingState]()
	n := NewNodes(cfg)

	if err := g.AddLambdaNode("AudioPreprocess", compose.InvokableLambda(
		n.PreprocessAudio,
	)); err != nil {
		return nil, fmt.Errorf("add AudioPreprocess node: %w", err)
	}

	if err := g.AddLambdaNode("ASR", compose.InvokableLambda(
		n.CallASR,
	)); err != nil {
		return nil, fmt.Errorf("add ASR node: %w", err)
	}

	if err := g.AddLambdaNode("Summarize", compose.InvokableLambda(
		n.Summarize,
	)); err != nil {
		return nil, fmt.Errorf("add Summarize node: %w", err)
	}

	if err := g.AddLambdaNode("FormatOutput", compose.InvokableLambda(
		n.FormatOutput,
	)); err != nil {
		return nil, fmt.Errorf("add FormatOutput node: %w", err)
	}

	if err := g.AddEdge(compose.START, "AudioPreprocess"); err != nil {
		return nil, fmt.Errorf("add edge START->AudioPreprocess: %w", err)
	}
	if err := g.AddEdge("AudioPreprocess", "ASR"); err != nil {
		return nil, fmt.Errorf("add edge AudioPreprocess->ASR: %w", err)
	}
	if err := g.AddEdge("ASR", "Summarize"); err != nil {
		return nil, fmt.Errorf("add edge ASR->Summarize: %w", err)
	}
	if err := g.AddEdge("Summarize", "FormatOutput"); err != nil {
		return nil, fmt.Errorf("add edge Summarize->FormatOutput: %w", err)
	}
	if err := g.AddEdge("FormatOutput", compose.END); err != nil {
		return nil, fmt.Errorf("add edge FormatOutput->END: %w", err)
	}

	runner, err := g.Compile(context.Background())
	if err != nil {
		return nil, fmt.Errorf("compile graph: %w", err)
	}

	return runner, nil
}
