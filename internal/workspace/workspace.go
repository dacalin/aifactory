package workspace

import (
	"context"

	"github.com/dacalin/aifactory/internal/model"
)

// Manager creates and removes task workspaces independently of task execution.
type Manager interface {
	Create(context.Context, model.Project, model.Task) (Workspace, error)
	Remove(context.Context, Workspace) error
}
