package samples

import (
	"context"
	"fmt"

	"github.com/JaimeStill/agent-lab/internal/workflows"
	"github.com/JaimeStill/go-agents-orchestration/pkg/state"
	"github.com/google/uuid"
)

func init() {
	workflows.Register("reasoning", reasoningFactory, "Multi-step reasoning workflow that analyzes problems")
}

func reasoningFactory(ctx context.Context, graph state.StateGraph, runtime *workflows.Runtime, params map[string]any) (state.State, error) {
	analyzeNode := state.NewFunctionNode(func(ctx context.Context, s state.State) (state.State, error) {
		agentID, token, err := extractAgentParams(s)
		if err != nil {
			return s, err
		}

		problem, ok := s.Get("problem")
		if !ok {
			return s, fmt.Errorf("problem is required")
		}

		systemPrompt := "You are an analytical assistant. Break down problems into their key components and identify the important elements."
		if sp, ok := s.Get("analyze_system_prompt"); ok {
			if spStr, ok := sp.(string); ok && spStr != "" {
				systemPrompt = spStr
			}
		}

		opts := map[string]any{
			"system_prompt": systemPrompt,
		}

		prompt := fmt.Sprintf("Analyze this problem and identify its key components:\n\n%s", problem)

		resp, err := runtime.Agents().Chat(ctx, agentID, prompt, opts, token)
		if err != nil {
			return s, fmt.Errorf("analyze failed: %w", err)
		}

		return s.Set("analysis", resp.Content()), nil
	})

	reasonNode := state.NewFunctionNode(func(ctx context.Context, s state.State) (state.State, error) {
		agentID, token, err := extractAgentParams(s)
		if err != nil {
			return s, err
		}

		analysis, ok := s.Get("analysis")
		if !ok {
			return s, fmt.Errorf("analysis not found in state")
		}

		systemPrompt := "You are a logical reasoning assistant. Think step-by-step and explain your reasoning clearly."
		if sp, ok := s.Get("reason_system_prompt"); ok {
			if spStr, ok := sp.(string); ok && spStr != "" {
				systemPrompt = spStr
			}
		}

		opts := map[string]any{
			"system_prompt": systemPrompt,
		}

		prompt := fmt.Sprintf("Given this analysis:\n\n%s\n\nWhat are the logical steps to solve this problem?", analysis)

		resp, err := runtime.Agents().Chat(ctx, agentID, prompt, opts, token)
		if err != nil {
			return s, fmt.Errorf("reason failed: %w", err)
		}

		return s.Set("reasoning", resp.Content()), nil
	})

	concludeNode := state.NewFunctionNode(func(ctx context.Context, s state.State) (state.State, error) {
		agentID, token, err := extractAgentParams(s)
		if err != nil {
			return s, err
		}

		reasoning, ok := s.Get("reasoning")
		if !ok {
			return s, fmt.Errorf("reasoning not found in state")
		}

		systemPrompt := "You are a concise assistant. Provide clear, direct conclusions based on the reasoning provided."
		if sp, ok := s.Get("conclude_system_prompt"); ok {
			if spStr, ok := sp.(string); ok && spStr != "" {
				systemPrompt = spStr
			}
		}

		opts := map[string]any{
			"system_prompt": systemPrompt,
		}

		prompt := fmt.Sprintf("Based on this reasoning:\n\n%s\n\nWhat is the conclusion?", reasoning)

		resp, err := runtime.Agents().Chat(ctx, agentID, prompt, opts, token)
		if err != nil {
			return s, fmt.Errorf("conclude failed: %w", err)
		}

		return s.Set("conclusion", resp.Content()), nil
	})

	if err := graph.AddNode("analyze", analyzeNode); err != nil {
		return state.State{}, err
	}

	if err := graph.AddNode("reason", reasonNode); err != nil {
		return state.State{}, err
	}

	if err := graph.AddNode("conclude", concludeNode); err != nil {
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

func extractAgentParams(s state.State) (uuid.UUID, string, error) {
	agentIDStr, ok := s.Get("agent_id")
	if !ok {
		return uuid.Nil, "", fmt.Errorf("agent_id is required")
	}

	agentID, err := uuid.Parse(agentIDStr.(string))
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("invalid agent_id: %w", err)
	}

	token, _ := s.Get("token")
	tokenStr, _ := token.(string)

	return agentID, tokenStr, nil
}
