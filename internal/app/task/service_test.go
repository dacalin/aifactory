package task

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/dacalin/aifactory/internal/model"
)

func TestCreateUsesProjectBranchByDefaultAndAllowsTaskOverride(t *testing.T) {
	projects := &projectsStub{project: model.Project{ID: "project-1", BaseBranch: "main"}}
	storage := &taskRepositoryStub{}
	service := NewService(projects, storage)
	service.clock = func() time.Time { return time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC) }

	created, err := service.Create(context.Background(), "project-1", "  Fix search  ", "  Details  ", "")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.BaseBranch != "main" || created.Title != "Fix search" || created.Description != "Details" {
		t.Fatalf("created task = %#v", created)
	}
	if created.Status != model.TaskStatusBacklog {
		t.Fatalf("status = %q, want %q", created.Status, model.TaskStatusBacklog)
	}

	overridden, err := service.Create(context.Background(), "project-1", "Add logging", "", "release/next")
	if err != nil {
		t.Fatalf("Create() with branch override error = %v", err)
	}
	if overridden.BaseBranch != "release/next" {
		t.Fatalf("task base branch = %q, want release/next", overridden.BaseBranch)
	}
}

func TestCreateRejectsUnknownTaskBaseBranch(t *testing.T) {
	service := NewService(&projectsStub{project: model.Project{ID: "project-1", BaseBranch: "main"}}, &taskRepositoryStub{})
	if _, err := service.Create(context.Background(), "project-1", "Task", "", "missing"); !errors.Is(err, ErrInvalidTask) {
		t.Fatalf("Create() error = %v, want ErrInvalidTask", err)
	}
}

type projectsStub struct {
	project model.Project
}

func (projects *projectsStub) Get(context.Context, string) (model.Project, error) {
	return projects.project, nil
}

func (*projectsStub) ValidateBranch(_ context.Context, _ string, branch string) error {
	if branch != "main" && branch != "release/next" {
		return errors.New("branch does not exist")
	}
	return nil
}

type taskRepositoryStub struct{}

func (*taskRepositoryStub) Create(_ context.Context, task model.Task) (model.Task, error) {
	return task, nil
}

func (*taskRepositoryStub) ListByProject(context.Context, string) ([]model.Task, error) {
	return nil, nil
}
