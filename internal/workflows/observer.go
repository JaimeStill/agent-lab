// observer.go provides PostgreSQL-backed event observation for workflow execution.
// It implements the observability.Observer interface from go-agents-orchestration.
package workflows

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/JaimeStill/agent-lab/pkg/decode"
	"github.com/JaimeStill/go-agents-orchestration/pkg/observability"
	"github.com/google/uuid"
)

// PostgresObserver implements observability.Observer using PostgreSQL.
// It records workflow execution events (stages and decisions) to the database
// for auditing and debugging purposes.
type PostgresObserver struct {
	db         *sql.DB
	runID      uuid.UUID
	logger     *slog.Logger
	mu         sync.Mutex
	startTimes map[string]time.Time
}

// NewPostgresObserver creates a new PostgreSQL-backed observer for a specific workflow run.
func NewPostgresObserver(db *sql.DB, runID uuid.UUID, logger *slog.Logger) *PostgresObserver {
	return &PostgresObserver{
		db:         db,
		runID:      runID,
		logger:     logger,
		startTimes: make(map[string]time.Time),
	}
}

// OnEvent handles workflow events, persisting relevant ones to the database.
// It handles EventNodeStart, EventNodeComplete, and EventEdgeTransition events.
func (o *PostgresObserver) OnEvent(ctx context.Context, event observability.Event) {
	o.mu.Lock()
	defer o.mu.Unlock()

	switch event.Type {
	case observability.EventNodeStart:
		o.handleNodeStart(ctx, event)
	case observability.EventNodeComplete:
		o.handleNodeComplete(ctx, event)
	case observability.EventEdgeTransition:
		o.handleEdgeTransition(ctx, event)
	default:
		o.logger.Debug("unhandled event", "type", event.Type, "source", event.Source)
	}
}

func (o *PostgresObserver) handleNodeStart(ctx context.Context, event observability.Event) {
	data, err := decode.FromMap[NodeStartData](event.Data)
	if err != nil {
		o.logger.Error("failed to decode node start data", "error", err)
		return
	}

	key := fmt.Sprintf("%s:%d", data.Node, data.Iteration)
	o.startTimes[key] = event.Timestamp

	var inputData []byte
	if data.InputSnapshot != nil {
		inputData, err = json.Marshal(data.InputSnapshot)
		if err != nil {
			o.logger.Error("failed to marshal input snapshot", "error", err, "node", data.Node)
		}
	}

	const query = `
		INSERT INTO stages (run_id, node_name, iteration, status, input_snapshot, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err = o.db.ExecContext(ctx, query, o.runID, data.Node, data.Iteration, StageStarted, inputData, event.Timestamp)
	if err != nil {
		o.logger.Error("failed to insert stage", "error", err, "node", data.Node)
	}
}

func (o *PostgresObserver) handleNodeComplete(ctx context.Context, event observability.Event) {
	data, err := decode.FromMap[NodeCompleteData](event.Data)
	if err != nil {
		o.logger.Error("failed to decode node complete data", "error", err)
		return
	}

	status := StageCompleted
	if data.Error {
		status = StageFailed
	}

	key := fmt.Sprintf("%s:%d", data.Node, data.Iteration)
	var durationMs *int
	if startTime, ok := o.startTimes[key]; ok {
		duration := int(event.Timestamp.Sub(startTime).Milliseconds())
		durationMs = &duration
		delete(o.startTimes, key)
	}

	var outputData []byte
	if data.OutputSnapshot != nil {
		outputData, err = json.Marshal(data.OutputSnapshot)
		if err != nil {
			o.logger.Error("failed to marshal output snapshot", "error", err, "node", data.Node)
		}
	}

	const query = `
		UPDATE stages
		SET status = $1, duration_ms = $2, output_snapshot = $3
		WHERE run_id = $4 AND node_name = $5 AND iteration = $6
	`

	_, err = o.db.ExecContext(ctx, query, status, durationMs, outputData, o.runID, data.Node, data.Iteration)
	if err != nil {
		o.logger.Error("failed to update stage", "error", err, "node", data.Node)
	}
}

func (o *PostgresObserver) handleEdgeTransition(ctx context.Context, event observability.Event) {
	data, err := decode.FromMap[EdgeTransitionData](event.Data)
	if err != nil {
		o.logger.Error("failed to decode edge transition data", "error", err)
		return
	}

	var predNamePtr *string
	if data.PredicateName != "" {
		predNamePtr = &data.PredicateName
	}

	const query = `
		INSERT INTO decisions (run_id, from_node, to_node, predicate_name, predicate_result, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err = o.db.ExecContext(ctx, query, o.runID, data.From, data.To, predNamePtr, data.PredicateResult, event.Timestamp)
	if err != nil {
		o.logger.Error("failed to insert decision", "error", err, "from", data.From, "to", data.To)
	}
}
