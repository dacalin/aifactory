package task

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/dacalin/aifactory/internal/model"
	"github.com/google/uuid"
)

type Service struct {
	projects   Projects
	repository Repository
	clock      func() time.Time
}

func NewService(projects Projects, repository Repository) *Service {
	return &Service{projects: projects, repository: repository, clock: func() time.Time { return time.Now().UTC() }}
}

func (service *Service) Create(ctx context.Context, projectID, title, description, requestedBranch string) (model.Task, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return model.Task{}, fmt.Errorf("%w: title is required", ErrInvalidTask)
	}
	project, err := service.projects.Get(ctx, projectID)
	if err != nil {
		return model.Task{}, err
	}
	baseBranch := strings.TrimSpace(requestedBranch)
	if baseBranch == "" {
		baseBranch = project.BaseBranch
	}
	if err := service.projects.ValidateBranch(ctx, projectID, baseBranch); err != nil {
		return model.Task{}, fmt.Errorf("%w: %v", ErrInvalidTask, err)
	}
	now := service.clock()
	task := model.Task{
		ID: uuid.NewString(), ProjectID: projectID, Title: title,
		Description: strings.TrimSpace(description), BaseBranch: baseBranch,
		Status: model.TaskStatusBacklog, CreatedAt: now, UpdatedAt: now,
	}
	return service.repository.Create(ctx, task)
}

func (service *Service) ListByProject(ctx context.Context, projectID string) ([]model.Task, error) {
	if _, err := service.projects.Get(ctx, projectID); err != nil {
		return nil, err
	}
	return service.repository.ListByProject(ctx, projectID)
}
