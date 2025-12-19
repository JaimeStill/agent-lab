package workflows

import (
	"context"

	"github.com/JaimeStill/agent-lab/pkg/pagination"
	"github.com/google/uuid"
)

// System defines the public interface for the workflows domain.
// It provides methods for listing, querying, and executing workflows.
type System interface {
	ListRuns(ctx context.Context, page pagination.PageRequest, filters RunFilters) (*pagination.PageResult[Run], error)
	FindRun(ctx context.Context, id uuid.UUID) (*Run, error)
	GetStages(ctx context.Context, runID uuid.UUID) ([]Stage, error)
	GetDecisions(ctx context.Context, runID uuid.UUID) ([]Decision, error)
	ListWorkflows() []WorkflowInfo
	Execute(ctx context.Context, name string, params map[string]any) (*Run, error)
	ExecuteStream(ctx context.Context, name string, params map[string]any) (<-chan ExecutionEvent, *Run, error)
	Cancel(ctx context.Context, runID uuid.UUID) error
	Resume(ctx context.Context, runID uuid.UUID) (*Run, error)
}
