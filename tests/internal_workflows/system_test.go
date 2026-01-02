package internal_workflows_test

import (
	"context"
	"testing"

	"github.com/JaimeStill/agent-lab/internal/workflows"
	"github.com/JaimeStill/agent-lab/pkg/pagination"
	"github.com/google/uuid"
)

func TestSystem_Interface(t *testing.T) {
	var sys workflows.System

	_ = sys

	t.Run("interface has expected methods", func(t *testing.T) {
		type systemInterface interface {
			ListRuns(ctx context.Context, page pagination.PageRequest, filters workflows.RunFilters) (*pagination.PageResult[workflows.Run], error)
			FindRun(ctx context.Context, id uuid.UUID) (*workflows.Run, error)
			GetStages(ctx context.Context, runID uuid.UUID) ([]workflows.Stage, error)
			GetDecisions(ctx context.Context, runID uuid.UUID) ([]workflows.Decision, error)
			DeleteRun(ctx context.Context, id uuid.UUID) error
			ListWorkflows() []workflows.WorkflowInfo
			Execute(ctx context.Context, name string, params map[string]any, token string) (*workflows.Run, error)
			ExecuteStream(ctx context.Context, name string, params map[string]any, token string) (<-chan workflows.ExecutionEvent, *workflows.Run, error)
			Cancel(ctx context.Context, runID uuid.UUID) error
			Resume(ctx context.Context, runID uuid.UUID) (*workflows.Run, error)
		}

		var _ systemInterface = (workflows.System)(nil)
	})
}
