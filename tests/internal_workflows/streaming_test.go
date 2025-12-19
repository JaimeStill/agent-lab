package internal_workflows_test

import (
	"context"
	"testing"
	"time"

	"github.com/JaimeStill/agent-lab/internal/workflows"
	"github.com/JaimeStill/go-agents-orchestration/pkg/observability"
)

type errorf string

func (e errorf) Error() string { return string(e) }
func TestStreamingObserver_Events(t *testing.T) {
	obs := workflows.NewStreamingObserver(10)
	events := obs.Events()

	if events == nil {
		t.Fatal("Events() returned nil channel")
	}
}

func TestStreamingObserver_SendComplete(t *testing.T) {
	obs := workflows.NewStreamingObserver(10)
	events := obs.Events()

	result := map[string]any{"output": "test"}
	obs.SendComplete(result)

	select {
	case event := <-events:
		if event.Type != workflows.EventComplete {
			t.Errorf("Type = %q, want %q", event.Type, workflows.EventComplete)
		}
		if event.Data["result"] == nil {
			t.Error("Data[result] is nil")
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timed out waiting for event")
	}
}

func TestStreamingObserver_SendError(t *testing.T) {
	obs := workflows.NewStreamingObserver(10)
	events := obs.Events()

	testErr := "test error"
	obs.SendError(errorf(testErr), "test-node")

	select {
	case event := <-events:
		if event.Type != workflows.EventError {
			t.Errorf("Type = %q, want %q", event.Type, workflows.EventError)
		}
		if event.Data["message"] != testErr {
			t.Errorf("Data[message] = %q, want %q", event.Data["message"], testErr)
		}
		if event.Data["node_name"] != "test-node" {
			t.Errorf("Data[node_name] = %q, want %q", event.Data["node_name"], "test-node")
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timed out waiting for event")
	}
}

func TestStreamingObserver_Close(t *testing.T) {
	obs := workflows.NewStreamingObserver(10)
	events := obs.Events()

	obs.Close()

	select {
	case _, ok := <-events:
		if ok {
			t.Error("Expected channel to be closed")
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timed out waiting for channel close")
	}
}

func TestStreamingObserver_CloseIdempotent(t *testing.T) {
	obs := workflows.NewStreamingObserver(10)

	obs.Close()
	obs.Close()
}

func TestStreamingObserver_SendAfterClose(t *testing.T) {
	obs := workflows.NewStreamingObserver(10)

	obs.Close()

	obs.SendComplete(map[string]any{})
	obs.SendError(errorf("test"), "")
}

func TestStreamingObserver_OnEvent_NodeStart(t *testing.T) {
	obs := workflows.NewStreamingObserver(10)
	events := obs.Events()

	event := observability.Event{
		Type:      observability.EventNodeStart,
		Timestamp: time.Now(),
		Data: map[string]any{
			"node":      "test-node",
			"iteration": 0,
		},
	}

	obs.OnEvent(context.Background(), event)

	select {
	case execEvent := <-events:
		if execEvent.Type != workflows.EventStageStart {
			t.Errorf("Type = %q, want %q", execEvent.Type, workflows.EventStageStart)
		}
		if execEvent.Data["node_name"] != "test-node" {
			t.Errorf("Data[node_name] = %q, want %q", execEvent.Data["node_name"], "test-node")
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timed out waiting for event")
	}
}

func TestStreamingObserver_OnEvent_NodeComplete(t *testing.T) {
	obs := workflows.NewStreamingObserver(10)
	events := obs.Events()

	event := observability.Event{
		Type:      observability.EventNodeComplete,
		Timestamp: time.Now(),
		Data: map[string]any{
			"node":      "test-node",
			"iteration": 1,
		},
	}

	obs.OnEvent(context.Background(), event)

	select {
	case execEvent := <-events:
		if execEvent.Type != workflows.EventStageComplete {
			t.Errorf("Type = %q, want %q", execEvent.Type, workflows.EventStageComplete)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timed out waiting for event")
	}
}

func TestStreamingObserver_OnEvent_NodeCompleteWithError(t *testing.T) {
	obs := workflows.NewStreamingObserver(10)
	events := obs.Events()

	event := observability.Event{
		Type:      observability.EventNodeComplete,
		Timestamp: time.Now(),
		Data: map[string]any{
			"node":          "test-node",
			"iteration":     0,
			"error":         true,
			"error_message": "node failed",
		},
	}

	obs.OnEvent(context.Background(), event)

	select {
	case execEvent := <-events:
		if execEvent.Type != workflows.EventError {
			t.Errorf("Type = %q, want %q", execEvent.Type, workflows.EventError)
		}
		if execEvent.Data["message"] != "node failed" {
			t.Errorf("Data[message] = %q, want %q", execEvent.Data["message"], "node failed")
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timed out waiting for event")
	}
}

func TestStreamingObserver_OnEvent_EdgeTransition(t *testing.T) {
	obs := workflows.NewStreamingObserver(10)
	events := obs.Events()

	event := observability.Event{
		Type:      observability.EventEdgeTransition,
		Timestamp: time.Now(),
		Data: map[string]any{
			"from":             "node-a",
			"to":               "node-b",
			"predicate_result": true,
		},
	}

	obs.OnEvent(context.Background(), event)

	select {
	case execEvent := <-events:
		if execEvent.Type != workflows.EventDecision {
			t.Errorf("Type = %q, want %q", execEvent.Type, workflows.EventDecision)
		}
		if execEvent.Data["from_node"] != "node-a" {
			t.Errorf("Data[from_node] = %q, want %q", execEvent.Data["from_node"], "node-a")
		}
		if execEvent.Data["to_node"] != "node-b" {
			t.Errorf("Data[to_node] = %q, want %q", execEvent.Data["to_node"], "node-b")
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timed out waiting for event")
	}
}
