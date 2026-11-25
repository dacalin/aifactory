package sqlite

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	projectdomain "github.com/dacalin/aifactory/internal/app/project"
	"github.com/dacalin/aifactory/internal/model"
)

func TestProjectRepositoryPersistsProjects(t *testing.T) {
	database, err := Open(filepath.Join(t.TempDir(), "factory.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	repository := NewProjectRepository(database.DB)
	projectModel := model.Project{ID: "project-1", Name: "Alpha", Path: "/projects/alpha", BaseBranch: "main", CreatedAt: time.Now().UTC()}

	created, wasCreated, err := repository.CreateOrGetByPath(context.Background(), projectModel)
	if err != nil {
		t.Fatalf("CreateOrGetByPath() error = %v", err)
	}
	if !wasCreated || created.ID != projectModel.ID {
		t.Fatalf("created project = %#v, created = %v", created, wasCreated)
	}
	if _, wasCreated, err = repository.CreateOrGetByPath(context.Background(), model.Project{ID: "duplicate", Name: "Renamed", Path: projectModel.Path}); err != nil || wasCreated {
		t.Fatalf("duplicate CreateOrGetByPath() created = %v, error = %v", wasCreated, err)
	}
	loaded, err := repository.Get(context.Background(), projectModel.ID)
	if err != nil || loaded.Path != projectModel.Path || loaded.BaseBranch != projectModel.BaseBranch {
		t.Fatalf("Get() project = %#v, error = %v", loaded, err)
	}
	if _, err := repository.Get(context.Background(), "missing"); !errors.Is(err, projectdomain.ErrProjectNotFound) {
		t.Fatalf("Get(missing) error = %v, want ErrProjectNotFound", err)
	}
	projects, err := repository.List(context.Background())
	if err != nil || len(projects) != 1 {
		t.Fatalf("List() projects = %#v, error = %v", projects, err)
	}
}

func TestTaskRepositoryPersistsSelectedBaseBranch(t *testing.T) {
	database, err := Open(filepath.Join(t.TempDir(), "factory.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	project := model.Project{ID: "project-1", Name: "Alpha", Path: "/projects/alpha", BaseBranch: "main", CreatedAt: time.Now().UTC()}
	if _, _, err := NewProjectRepository(database.DB).CreateOrGetByPath(context.Background(), project); err != nil {
		t.Fatalf("CreateOrGetByPath() error = %v", err)
	}
	task := model.Task{
		ID: "task-1", ProjectID: project.ID, Title: "Fix search", Description: "", BaseBranch: "release/next",
		Status: model.TaskStatusBacklog, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	repository := NewTaskRepository(database.DB)
	if _, err := repository.Create(context.Background(), task); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	tasks, err := repository.ListByProject(context.Background(), project.ID)
	if err != nil || len(tasks) != 1 {
		t.Fatalf("ListByProject() tasks = %#v, error = %v", tasks, err)
	}
	if tasks[0].BaseBranch != "release/next" {
		t.Fatalf("task base branch = %q, want release/next", tasks[0].BaseBranch)
	}
}
