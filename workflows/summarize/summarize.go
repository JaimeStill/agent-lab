package summarize

import (
	"context"
	"fmt"

	"github.com/JaimeStill/agent-lab/internal/workflows"
	"github.com/JaimeStill/go-agents-orchestration/pkg/state"
)

func init() {
	workflows.Register("summarize", factory, "Summarizes input text using an AI agent")
}

func factory(ctx context.Context, graph state.StateGraph, runtime *workflows.Runtime, params map[string]any) (state.State, error) {
	profile, err := workflows.LoadProfile(ctx, runtime, params, DefaultProfile())
	if err != nil {
		return state.State{}, err
	}

	summarizeNode := state.NewFunctionNode(func(ctx context.Context, s state.State) (state.State, error) {
		stage := profile.Stage("summarize")

		agentID, token, err := workflows.ExtractAgentParams(s, stage)
		if err != nil {
			return s, err
		}

		text, ok := s.Get("text")
		if !ok {
			return s, fmt.Errorf("text is required")
		}

		opts := map[string]any{
			"system_prompt": *stage.SystemPrompt,
		}

		prompt := fmt.Sprintf("Please summarize the following text:\n\n%s", text)

		resp, err := runtime.Agents().Chat(ctx, agentID, prompt, opts, token)
		if err != nil {
			return s, fmt.Errorf("chat failed: %w", err)
		}

		return s.Set("summary", resp.Content()), nil
	})

	if err := graph.AddNode("summarize", summarizeNode); err != nil {
		return state.State{}, err
	}

	if err := graph.SetEntryPoint("summarize"); err != nil {
		return state.State{}, err
	}

	if err := graph.SetExitPoint("summarize"); err != nil {
		return state.State{}, err
	}

	initialState := state.New(nil)
	for k, v := range params {
		initialState = initialState.Set(k, v)
	}

	return initialState, nil
}
