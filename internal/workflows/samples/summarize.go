// Package samples provides example workflow implementations demonstrating
// live agent integration with the workflow execution infrastructure.
package samples

import (
	"context"
	"fmt"

	"github.com/JaimeStill/agent-lab/internal/workflows"
	"github.com/JaimeStill/go-agents-orchestration/pkg/state"
	"github.com/google/uuid"
)

func init() {
	workflows.Register("summarize", summarizeFactory, "Summarizes input text using an AI agent")
}

func summarizeFactory(ctx context.Context, graph state.StateGraph, runtime *workflows.Runtime, params map[string]any) (state.State, error) {
	summarizeNode := state.NewFunctionNode(func(ctx context.Context, s state.State) (state.State, error) {
		agentIDStr, ok := s.Get("agent_id")
		if !ok {
			return s, fmt.Errorf("agent_id is required")
		}

		agentID, err := uuid.Parse(agentIDStr.(string))
		if err != nil {
			return s, fmt.Errorf("invalid agent_id: %w", err)
		}

		text, ok := s.Get("text")
		if !ok {
			return s, fmt.Errorf("text is required")
		}

		token, _ := s.Get("token")
		tokenStr, _ := token.(string)

		systemPrompt := "You are a concise summarization assistant. Provide clear, brief summaries that capture the key points."
		if sp, ok := s.Get("system_prompt"); ok {
			if spStr, ok := sp.(string); ok && spStr != "" {
				systemPrompt = spStr
			}
		}

		opts := map[string]any{
			"system_prompt": systemPrompt,
		}

		prompt := fmt.Sprintf("Please summarize the following text:\n\n%s", text)

		resp, err := runtime.Agents().Chat(ctx, agentID, prompt, opts, tokenStr)
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
