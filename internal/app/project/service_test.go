package project

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/dacalin/aifactory/internal/model"
)

func TestCreateProjectUsesSourceControlAndPersistsProject(t *testing.T) {
	store := &projectStoreStub{}
	source := &sourceControlStub{createdPath: "/projects/alpha"}
	service := NewService(store, source)
	service.clock = func() time.Time { return time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC) }

	created, wasCreated, err := service.Create(context.Background(), " Alpha ", "/projects")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if !wasCreated || created.Name != "Alpha" || created.Path != "/projects/alpha" || created.BaseBranch != "main" {
		t.Fatalf("Create() = %#v, created = %v", created, wasCreated)
	}
	if source.createPath != "/projects" || store.created.Path != "/projects/alpha" {
		t.Fatalf("source path = %q, stored path = %q", source.createPath, store.created.Path)
	}
}

func TestCreateRejectsInvalidProjectNames(t *testing.T) {
	service := NewService(&projectStoreStub{}, &sourceControlStub{})
	for _, name := range []string{"", "..", ".git", "nested/name"} {
		if _, _, err := service.Create(context.Background(), name, "/projects"); !errors.Is(err, ErrInvalidProject) {
			t.Errorf("Create(%q) error = %v, want ErrInvalidProject", name, err)
		}
	}
}

type projectStoreStub struct {
	created model.Project
}

func (store *projectStoreStub) CreateOrGetByPath(_ context.Context, project model.Project) (model.Project, bool, error) {
	store.created = project
	return project, true, nil
}

func (*projectStoreStub) Get(context.Context, string) (model.Project, error) {
	return model.Project{}, nil
}
func (*projectStoreStub) List(context.Context) ([]model.Project, error) { return nil, nil }

type sourceControlStub struct {
	createPath  string
	createdPath string
}

func (source *sourceControlStub) CreateProject(_ context.Context, path string) (string, error) {
	source.createPath = path
	return source.createdPath, nil
}

func (*sourceControlStub) ResolveProject(context.Context, string) (string, error) {
	return "", nil
}

func (*sourceControlStub) DefaultBranch(context.Context, string) (string, error) {
	return "main", nil
}

func (*sourceControlStub) ListBranches(context.Context, string) ([]string, error) {
	return []string{"main", "origin/main"}, nil
}

func (*sourceControlStub) BranchExists(_ context.Context, _ string, branch string) (bool, error) {
	return branch == "main" || branch == "origin/main", nil
}

func (store *projectStoreStub) UpdateBaseBranch(_ context.Context, id, branch string) (model.Project, error) {
	if store.created.ID != id {
		return model.Project{}, errors.New("project not found")
	}
	store.created.BaseBranch = branch
	return store.created, nil
}
