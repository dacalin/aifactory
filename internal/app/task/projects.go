package task

import (
	"context"

	"github.com/dacalin/aifactory/internal/model"
)

type Projects interface {
	Get(context.Context, string) (model.Project, error)
	ValidateBranch(context.Context, string, string) error
}
