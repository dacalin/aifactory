package server

import (
	"context"

	"github.com/dacalin/aifactory/internal/model"
)

type Projects interface {
	Create(context.Context, string, string) (model.Project, bool, error)
	Open(context.Context, string) (model.Project, bool, error)
	Get(context.Context, string) (model.Project, error)
	List(context.Context) ([]model.Project, error)
	ListBranches(context.Context, string) ([]string, error)
	SetBaseBranch(context.Context, string, string) (model.Project, error)
}
