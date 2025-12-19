package workflows

import (
	"context"
	"sync"
	"time"

	"github.com/JaimeStill/agent-lab/pkg/decode"
	"github.com/JaimeStill/go-agents-orchestration/pkg/observability"
)

// MultiObserver broadcasts events to multiple observers.
// It implements observability.Observer and forwards events to all wrapped observers.
type MultiObserver struct {
	observers []observability.Observer
}

// NewMultiObserver creates a MultiObserver that broadcasts to all provided observers.
func NewMultiObserver(observers ...observability.Observer) *MultiObserver {
	return &MultiObserver{observers: observers}
}

// OnEvent forwards the event to all wrapped observers.
func (m *MultiObserver) OnEvent(ctx context.Context, event observability.Event) {
	for _, obs := range m.observers {
		obs.OnEvent(ctx, event)
	}
}

// StreamingObserver converts graph execution events to ExecutionEvents
// and sends them to a buffered channel for SSE streaming.
type StreamingObserver struct {
	events chan ExecutionEvent
	mu     sync.Mutex
	closed bool
}

// NewStreamingObserver creates a StreamingObserver with the specified buffer size.
func NewStreamingObserver(bufferSize int) *StreamingObserver {
	return &StreamingObserver{
		events: make(chan ExecutionEvent, bufferSize),
	}
}

// Events returns a read-only channel for consuming execution events.
func (o *StreamingObserver) Events() <-chan ExecutionEvent {
	return o.events
}

// Close closes the event channel. Safe to call multiple times.
func (o *StreamingObserver) Close() {
	o.mu.Lock()
	defer o.mu.Unlock()
	if !o.closed {
		o.closed = true
		close(o.events)
	}
}

// OnEvent handles observability events and converts them to ExecutionEvents.
func (o *StreamingObserver) OnEvent(ctx context.Context, event observability.Event) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.closed {
		return
	}

	var execEvent *ExecutionEvent

	switch event.Type {
	case observability.EventNodeStart:
		execEvent = o.handleNodeStart(event)
	case observability.EventNodeComplete:
		execEvent = o.handleNodeComplete(event)
	case observability.EventEdgeTransition:
		execEvent = o.handleEdgeTransition(event)
	}

	if execEvent != nil {
		select {
		case o.events <- *execEvent:
		default:
		}
	}
}

// SendComplete sends a completion event with the workflow result.
func (o *StreamingObserver) SendComplete(result map[string]any) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.closed {
		return
	}
	select {
	case o.events <- ExecutionEvent{
		Type:      EventComplete,
		Timestamp: time.Now(),
		Data:      map[string]any{"result": result},
	}:
	default:
	}
}

// SendError sends an error event with the error message and optional node name.
func (o *StreamingObserver) SendError(err error, nodeName string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.closed {
		return
	}
	data := map[string]any{"message": err.Error()}
	if nodeName != "" {
		data["node_name"] = nodeName
	}
	select {
	case o.events <- ExecutionEvent{
		Type:      EventError,
		Timestamp: time.Now(),
		Data:      data,
	}:
	default:
	}
}

func (o *StreamingObserver) handleNodeStart(event observability.Event) *ExecutionEvent {
	data, err := decode.FromMap[NodeStartData](event.Data)
	if err != nil {
		return nil
	}
	return &ExecutionEvent{
		Type:      EventStageStart,
		Timestamp: event.Timestamp,
		Data: map[string]any{
			"node_name": data.Node,
			"iteration": data.Iteration,
		},
	}
}

func (o *StreamingObserver) handleNodeComplete(event observability.Event) *ExecutionEvent {
	data, err := decode.FromMap[NodeCompleteData](event.Data)
	if err != nil {
		return nil
	}
	if data.Error {
		return &ExecutionEvent{
			Type:      EventError,
			Timestamp: event.Timestamp,
			Data: map[string]any{
				"node_name": data.Node,
				"message":   data.ErrorMessage,
			},
		}
	}
	return &ExecutionEvent{
		Type:      EventStageComplete,
		Timestamp: event.Timestamp,
		Data: map[string]any{
			"node_name":       data.Node,
			"iteration":       data.Iteration,
			"output_snapshot": data.OutputSnapshot,
		},
	}
}

func (o *StreamingObserver) handleEdgeTransition(event observability.Event) *ExecutionEvent {
	data, err := decode.FromMap[EdgeTransitionData](event.Data)
	if err != nil {
		return nil
	}
	return &ExecutionEvent{
		Type:      EventDecision,
		Timestamp: event.Timestamp,
		Data: map[string]any{
			"from_node":        data.From,
			"to_node":          data.To,
			"predicate_result": data.PredicateResult,
		},
	}
}
