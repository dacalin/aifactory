package project

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/dacalin/aifactory/internal/model"
	"github.com/dacalin/aifactory/internal/workspace"
	"github.com/google/uuid"
)

var ErrInvalidProject = errors.New("invalid project")

type Service struct {
	store         Repository
	sourceControl workspace.SourceControl
	clock         func() time.Time
}

func NewService(store Repository, sourceControl workspace.SourceControl) *Service {
	return &Service{store: store, sourceControl: sourceControl, clock: func() time.Time { return time.Now().UTC() }}
}

func (service *Service) Create(ctx context.Context, name, requestedPath string) (model.Project, bool, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return model.Project{}, false, fmt.Errorf("%w: project name is required", ErrInvalidProject)
	}
	if name == "." || name == ".." || strings.EqualFold(name, ".git") || strings.ContainsAny(name, `/\\`) || strings.ContainsRune(name, 0) {
		return model.Project{}, false, fmt.Errorf("%w: project name must be a single folder name", ErrInvalidProject)
	}
	path, err := service.sourceControl.CreateProject(ctx, requestedPath)
	if err != nil {
		return model.Project{}, false, err
	}
	baseBranch, err := service.sourceControl.DefaultBranch(ctx, path)
	if err != nil {
		return model.Project{}, false, fmt.Errorf("resolve project base branch: %w", err)
	}
	return service.store.CreateOrGetByPath(ctx, model.Project{
		ID: uuid.NewString(), Name: name, Path: path, BaseBranch: baseBranch, CreatedAt: service.clock(),
	})
}

func (service *Service) Open(ctx context.Context, requestedPath string) (model.Project, bool, error) {
	path, err := service.sourceControl.ResolveProject(ctx, requestedPath)
	if err != nil {
		return model.Project{}, false, err
	}
	baseBranch, err := service.sourceControl.DefaultBranch(ctx, path)
	if err != nil {
		return model.Project{}, false, fmt.Errorf("resolve project base branch: %w", err)
	}
	name := filepath.Base(path)
	if name == string(filepath.Separator) || name == "." || name == "" {
		return model.Project{}, false, fmt.Errorf("%w: filesystem root cannot be opened as a project", ErrInvalidProject)
	}
	return service.store.CreateOrGetByPath(ctx, model.Project{
		ID: uuid.NewString(), Name: name, Path: path, BaseBranch: baseBranch, CreatedAt: service.clock(),
	})
}

func (service *Service) Get(ctx context.Context, id string) (model.Project, error) {
	project, err := service.store.Get(ctx, id)
	if err != nil {
		return model.Project{}, err
	}
	return service.ensureBaseBranch(ctx, project)
}

func (service *Service) List(ctx context.Context) ([]model.Project, error) {
	projects, err := service.store.List(ctx)
	if err != nil {
		return nil, err
	}
	for index, project := range projects {
		projects[index], err = service.ensureBaseBranch(ctx, project)
		if err != nil {
			return nil, fmt.Errorf("load base branch for project %s: %w", project.ID, err)
		}
	}
	return projects, nil
}

func (service *Service) ensureBaseBranch(ctx context.Context, project model.Project) (model.Project, error) {
	if project.BaseBranch != "" {
		return project, nil
	}
	branch, err := service.sourceControl.DefaultBranch(ctx, project.Path)
	if err != nil {
		return model.Project{}, fmt.Errorf("resolve project base branch: %w", err)
	}
	return service.store.UpdateBaseBranch(ctx, project.ID, branch)
}

func (service *Service) ListBranches(ctx context.Context, id string) ([]string, error) {
	project, err := service.store.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return service.sourceControl.ListBranches(ctx, project.Path)
}

func (service *Service) SetBaseBranch(ctx context.Context, id, branch string) (model.Project, error) {
	if err := service.ValidateBranch(ctx, id, branch); err != nil {
		return model.Project{}, err
	}
	return service.store.UpdateBaseBranch(ctx, id, strings.TrimSpace(branch))
}

func (service *Service) ValidateBranch(ctx context.Context, id, branch string) error {
	branch = strings.TrimSpace(branch)
	if branch == "" {
		return errors.New("base branch is required")
	}
	project, err := service.store.Get(ctx, id)
	if err != nil {
		return err
	}
	exists, err := service.sourceControl.BranchExists(ctx, project.Path, branch)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("branch %q does not exist in project repository", branch)
	}
	return nil
}
