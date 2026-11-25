package task

import (
	"context"

	"github.com/dacalin/aifactory/internal/model"
)

type Repository interface {
	Create(context.Context, model.Task) (model.Task, error)
	ListByProject(context.Context, string) ([]model.Task, error)
}
