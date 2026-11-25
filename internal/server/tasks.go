package server

import (
	"context"

	"github.com/dacalin/aifactory/internal/model"
)

type Tasks interface {
	Create(context.Context, string, string, string, string) (model.Task, error)
	ListByProject(context.Context, string) ([]model.Task, error)
}
