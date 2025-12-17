// checkpoint.go provides PostgreSQL-backed checkpoint storage for workflow state persistence.
// It implements the state.CheckpointStore interface from go-agents-orchestration.
package workflows

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/JaimeStill/agent-lab/pkg/repository"
	"github.com/JaimeStill/go-agents-orchestration/pkg/state"
)

// PostgresCheckpointStore implements state.CheckpointStore using PostgreSQL.
// It stores workflow state as JSON for persistence and recovery.
type PostgresCheckpointStore struct {
	db     *sql.DB
	logger *slog.Logger
}

// NewPostgresCheckpointStore creates a new PostgreSQL-backed checkpoint store.
func NewPostgresCheckpointStore(db *sql.DB, logger *slog.Logger) *PostgresCheckpointStore {
	return &PostgresCheckpointStore{
		db:     db,
		logger: logger,
	}
}

// Save persists workflow state to the database. It uses upsert semantics,
// updating existing checkpoints for the same run_id.
func (s *PostgresCheckpointStore) Save(st state.State) error {
	stateData, err := json.Marshal(st)
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}

	const query = `
		INSERT INTO checkpoints (run_id, state_data, checkpoint_node, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
		ON CONFLICT (run_id) DO UPDATE SET
			state_data = EXCLUDED.state_data,
			checkpoint_node = EXCLUDED.checkpoint_node,
			updated_at = NOW()
	`

	_, err = s.db.ExecContext(
		context.Background(),
		query,
		st.RunID,
		stateData,
		st.CheckpointNode,
	)

	if err != nil {
		return fmt.Errorf("save checkpoint: %w", err)
	}

	s.logger.Debug("checkpoint saved", "run_id", st.RunID, "node", st.CheckpointNode)
	return nil
}

// Load retrieves workflow state from the database by run ID.
// Returns an error if the checkpoint is not found.
func (s *PostgresCheckpointStore) Load(runID string) (state.State, error) {
	const query = `SELECT state_data FROM checkpoints WHERE run_id = $1`

	var stateData []byte
	err := s.db.QueryRowContext(context.Background(), query, runID).Scan(&stateData)
	if err != nil {
		if err == sql.ErrNoRows {
			return state.State{}, fmt.Errorf("checkpoint not found: %s", runID)
		}
		return state.State{}, fmt.Errorf("query checkpoint: %w", err)
	}

	var st state.State
	if err := json.Unmarshal(stateData, &st); err != nil {
		return state.State{}, fmt.Errorf("unmarshal state: %w", err)
	}

	s.logger.Debug("checkpoint loaded", "run_id", runID)
	return st, nil
}

// Delete removes a checkpoint from the database by run ID.
func (s *PostgresCheckpointStore) Delete(runID string) error {
	const query = `DELETE FROM checkpoints WHERE run_id = $1`

	_, err := s.db.ExecContext(context.Background(), query, runID)
	if err != nil {
		return fmt.Errorf("delete checkpoint: %w", err)
	}

	s.logger.Debug("checkpoint deleted", "run_id", runID)
	return nil
}

// List returns all checkpoint run IDs ordered by creation time (newest first).
func (s *PostgresCheckpointStore) List() ([]string, error) {
	const query = `SELECT run_id FROM checkpoints ORDER BY created_at DESC`

	ids, err := repository.QueryMany(context.Background(), s.db, query, nil, func(sc repository.Scanner) (string, error) {
		var id string
		err := sc.Scan(&id)
		return id, err
	})
	if err != nil {
		return nil, fmt.Errorf("list checkpoints: %w", err)
	}

	return ids, nil
}
