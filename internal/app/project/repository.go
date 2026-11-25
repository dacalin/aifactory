package project

import (
	"context"

	"github.com/dacalin/aifactory/internal/model"
)

// Repository owns project persistence required by the project service.
type Repository interface {
	CreateOrGetByPath(context.Context, model.Project) (project model.Project, created bool, err error)
	Get(context.Context, string) (model.Project, error)
	List(context.Context) ([]model.Project, error)
	UpdateBaseBranch(context.Context, string, string) (model.Project, error)
}
