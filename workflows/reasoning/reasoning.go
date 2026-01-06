package reasoning

import (
	"context"
	"fmt"

	"github.com/JaimeStill/agent-lab/internal/profiles"
	"github.com/JaimeStill/agent-lab/internal/workflows"
	"github.com/JaimeStill/go-agents-orchestration/pkg/state"
)

func init() {
	workflows.Register("reasoning", factory, "Multi-step reasoning workflow that analyzes problems")
}

func factory(ctx context.Context, graph state.StateGraph, runtime *workflows.Runtime, params map[string]any) (state.State, error) {
	profile, err := workflows.LoadProfile(ctx, runtime, params, DefaultProfile())
	if err != nil {
		return state.State{}, err
	}

	if err := graph.AddNode("analyze", analyzeNode(profile, runtime)); err != nil {
		return state.State{}, err
	}

	if err := graph.AddNode("reason", reasonNode(profile, runtime)); err != nil {
		return state.State{}, err
	}

	if err := graph.AddNode("conclude", concludeNode(profile, runtime)); err != nil {
		return state.State{}, err
	}

	if err := graph.AddEdge("analyze", "reason", nil); err != nil {
		return state.State{}, err
	}

	if err := graph.AddEdge("reason", "conclude", nil); err != nil {
		return state.State{}, err
	}

	if err := graph.SetEntryPoint("analyze"); err != nil {
		return state.State{}, err
	}

	if err := graph.SetExitPoint("conclude"); err != nil {
		return state.State{}, err
	}

	initialState := state.New(nil)
	for k, v := range params {
		initialState = initialState.Set(k, v)
	}

	return initialState, nil
}

func analyzeNode(profile *profiles.ProfileWithStages, runtime *workflows.Runtime) state.StateNode {
	return state.NewFunctionNode(func(ctx context.Context, s state.State) (state.State, error) {
		stage := profile.Stage("analyze")
		agentID, token, err := workflows.ExtractAgentParams(s, stage)
		if err != nil {
			return s, err
		}

		problem, ok := s.Get("problem")
		if !ok {
			return s, fmt.Errorf("problem is required")
		}

		opts := map[string]any{
			"system_prompt": *stage.SystemPrompt,
		}

		prompt := fmt.Sprintf("Analyze this problem and identify its key components:\n\n%s", problem)

		resp, err := runtime.Agents().Chat(ctx, agentID, prompt, opts, token)
		if err != nil {
			return s, fmt.Errorf("analyze failed: %w", err)
		}

		return s.Set("analysis", resp.Content()), nil
	})
}

func reasonNode(profile *profiles.ProfileWithStages, runtime *workflows.Runtime) state.StateNode {
	return state.NewFunctionNode(func(ctx context.Context, s state.State) (state.State, error) {
		stage := profile.Stage("reason")

		agentID, token, err := workflows.ExtractAgentParams(s, stage)
		if err != nil {
			return s, err
		}

		analysis, ok := s.Get("analysis")
		if !ok {
			return s, fmt.Errorf("analysis not found in state")
		}

		opts := map[string]any{
			"system_prompt": *stage.SystemPrompt,
		}

		prompt := fmt.Sprintf("Given this analysis:\n\n%s\n\nWhat are the logical steps to solve this problem?", analysis)

		resp, err := runtime.Agents().Chat(ctx, agentID, prompt, opts, token)
		if err != nil {
			return s, fmt.Errorf("reason failed: %w", err)
		}

		return s.Set("reasoning", resp.Content()), nil
	})
}

func concludeNode(profile *profiles.ProfileWithStages, runtime *workflows.Runtime) state.StateNode {
	return state.NewFunctionNode(func(ctx context.Context, s state.State) (state.State, error) {
		stage := profile.Stage("conclude")

		agentID, token, err := workflows.ExtractAgentParams(s, stage)
		if err != nil {
			return s, err
		}

		reasoning, ok := s.Get("reasoning")
		if !ok {
			return s, fmt.Errorf("reasoning not found in state")
		}

		opts := map[string]any{
			"system_prompt": *stage.SystemPrompt,
		}

		prompt := fmt.Sprintf("Based on this reasoning:\n\n%s\n\nWhat is the conclusion?", reasoning)

		resp, err := runtime.Agents().Chat(ctx, agentID, prompt, opts, token)
		if err != nil {
			return s, fmt.Errorf("conclude failed: %w", err)
		}

		return s.Set("conclusion", resp.Content()), nil
	})
}
