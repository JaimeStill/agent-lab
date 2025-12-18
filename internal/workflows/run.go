// Package workflows provides workflow execution infrastructure including
// run tracking, stage observation, and decision logging.
package workflows

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type ExecutionEventType string

const (
	EventStageStart    ExecutionEventType = "stage.start"
	EventStageComplete ExecutionEventType = "stage.complete"
	EventDecision      ExecutionEventType = "decision"
	EventError         ExecutionEventType = "error"
	EventComplete      ExecutionEventType = "complete"
)

type ExecutionEvent struct {
	Type      ExecutionEventType `json:"type"`
	Timestamp time.Time          `json:"timestamp"`
	Data      map[string]any     `json:"data"`
}

// RunStatus represents the execution state of a workflow run.
type RunStatus string

// Run status constants.
const (
	StatusPending   RunStatus = "pending"
	StatusRunning   RunStatus = "running"
	StatusCompleted RunStatus = "completed"
	StatusFailed    RunStatus = "failed"
	StatusCancelled RunStatus = "cancelled"
)

// StageStatus represents the execution state of a workflow stage.
type StageStatus string

// Stage status constants.
const (
	StageStarted   StageStatus = "started"
	StageCompleted StageStatus = "completed"
	StageFailed    StageStatus = "failed"
)

// Run represents a workflow execution record.
type Run struct {
	ID           uuid.UUID       `json:"id"`
	WorkflowName string          `json:"workflow_name"`
	Status       RunStatus       `json:"status"`
	Params       json.RawMessage `json:"params,omitempty"`
	Result       json.RawMessage `json:"result,omitempty"`
	ErrorMessage *string         `json:"error_message,omitempty"`
	StartedAt    *time.Time      `json:"started_at,omitempty"`
	CompletedAt  *time.Time      `json:"completed_at,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

// Stage represents a node execution within a workflow run.
type Stage struct {
	ID             uuid.UUID       `json:"id"`
	RunID          uuid.UUID       `json:"run_id"`
	NodeName       string          `json:"node_name"`
	Iteration      int             `json:"iteration"`
	Status         StageStatus     `json:"status"`
	InputSnapshot  json.RawMessage `json:"input_snapshot,omitempty"`
	OutputSnapshot json.RawMessage `json:"output_snapshot,omitempty"`
	DurationMs     *int            `json:"duration_ms,omitempty"`
	ErrorMessage   *string         `json:"error_message,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
}

// Decision represents a routing decision made during workflow execution.
type Decision struct {
	ID              uuid.UUID `json:"id"`
	RunID           uuid.UUID `json:"run_id"`
	FromNode        string    `json:"from_node"`
	ToNode          *string   `json:"to_node,omitempty"`
	PredicateName   *string   `json:"predicate_name,omitempty"`
	PredicateResult *bool     `json:"predicate_result,omitempty"`
	Reason          *string   `json:"reason,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

// WorkflowInfo provides metadata about a registered workflow.
type WorkflowInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// NodeStartData represents the data payload for node start events
// from go-agents-orchestration's observability.Event.
type NodeStartData struct {
	Node          string         `json:"node"`
	Iteration     int            `json:"iteration"`
	InputSnapshot map[string]any `json:"input_snapshot,omitempty"`
}

// NodeCompleteData represents the data payload for node complete events
// from go-agents-orchestration's observability.Event.
type NodeCompleteData struct {
	Node           string         `json:"node"`
	Iteration      int            `json:"iteration"`
	OutputSnapshot map[string]any `json:"output_snapshot,omitempty"`
	Error          bool           `json:"error,omitempty"`
	ErrorMessage   string         `json:"error_message,omitempty"`
}

// EdgeTransitionData represents the data payload for edge transition events
// from go-agents-orchestration's observability.Event.
type EdgeTransitionData struct {
	From            string `json:"from"`
	To              string `json:"to"`
	PredicateName   string `json:"predicate_name,omitempty"`
	PredicateResult bool   `json:"predicate_result"`
}
