package git

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dacalin/aifactory/internal/model"
	"github.com/dacalin/aifactory/internal/workspace"
)

func TestManagerCreatesReusesAndRemovesTaskWorktree(t *testing.T) {
	repository := initTestRepository(t)
	manager := NewManager(filepath.Join(t.TempDir(), "workspaces"))
	baseBranch := gitOutputForTest(t, repository, "branch", "--show-current")
	project := model.Project{ID: "project-1", Path: repository, BaseBranch: baseBranch}
	task := model.Task{ID: "task-1", ProjectID: project.ID, BaseBranch: baseBranch}

	t.Setenv("GIT_INDEX_FILE", ".git/index")
	created, err := manager.Create(context.Background(), project, task)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.Branch != "aifactory/project-1/task-1" {
		t.Fatalf("branch = %q", created.Branch)
	}
	if created.Path == repository {
		t.Fatal("task workspace points at the source repository")
	}
	if _, err := os.Stat(filepath.Join(created.Path, "README.md")); err != nil {
		t.Fatalf("expected project files in worktree: %v", err)
	}

	reused, err := manager.Create(context.Background(), project, task)
	if err != nil {
		t.Fatalf("Create() on existing task workspace error = %v", err)
	}
	if reused.Path != created.Path {
		t.Fatalf("reused path = %q, want %q", reused.Path, created.Path)
	}

	if err := manager.Remove(context.Background(), created); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
	if _, err := os.Stat(created.Path); !os.IsNotExist(err) {
		t.Fatalf("workspace still exists after removal: %v", err)
	}
}

func TestManagerCreatesWorktreeFromTaskBaseBranch(t *testing.T) {
	repository := initTestRepository(t)
	mainBranch := gitOutputForTest(t, repository, "branch", "--show-current")
	runGit(t, repository, "checkout", "-b", "release/next")
	if err := os.WriteFile(filepath.Join(repository, "release-marker.txt"), []byte("release\n"), 0600); err != nil {
		t.Fatal(err)
	}
	runGit(t, repository, "add", "release-marker.txt")
	runGit(t, repository, "commit", "-m", "release base")
	runGit(t, repository, "checkout", mainBranch)

	project := model.Project{ID: "project-2", Path: repository, BaseBranch: mainBranch}
	task := model.Task{ID: "task-2", ProjectID: project.ID, BaseBranch: "release/next"}
	manager := NewManager(filepath.Join(t.TempDir(), "workspaces"))
	created, err := manager.Create(context.Background(), project, task)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.BaseBranch != "release/next" {
		t.Fatalf("workspace base branch = %q, want release/next", created.BaseBranch)
	}
	if _, err := os.Stat(filepath.Join(created.Path, "release-marker.txt")); err != nil {
		t.Fatalf("worktree does not contain task base branch contents: %v", err)
	}
}

func TestManagerRejectsUnsafeIdentifiersAndPaths(t *testing.T) {
	repository := initTestRepository(t)
	manager := NewManager(filepath.Join(t.TempDir(), "workspaces"))
	if _, err := manager.Create(context.Background(), model.Project{ID: "../escape", Path: repository}, model.Task{ID: "task", ProjectID: "../escape", BaseBranch: "main"}); err == nil {
		t.Fatal("Create() accepted an unsafe project ID")
	}
	if err := manager.Remove(context.Background(), workspaceFixture(t, repository, t.TempDir())); err == nil {
		t.Fatal("Remove() accepted a path outside the workspace root")
	}
}

func workspaceFixture(t *testing.T, repository, path string) workspace.Workspace {
	t.Helper()
	return workspace.Workspace{ProjectPath: repository, Path: path}
}

func initTestRepository(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	runGit(t, root, "init")
	runGit(t, root, "config", "user.email", "test@example.com")
	runGit(t, root, "config", "user.name", "AI Factory Test")
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("test\n"), 0600); err != nil {
		t.Fatal(err)
	}
	runGit(t, root, "add", "README.md")
	runGit(t, root, "commit", "-m", "initial")
	return root
}

func runGit(t *testing.T, directory string, args ...string) {
	t.Helper()
	if output, err := combinedOutput(t.Context(), directory, args...); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, output)
	}
}

func gitOutputForTest(t *testing.T, directory string, args ...string) string {
	t.Helper()
	output, err := outputBytes(t.Context(), directory, args...)
	if err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
	return strings.TrimSpace(string(output))
}
