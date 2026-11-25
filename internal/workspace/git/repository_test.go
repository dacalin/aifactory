package git

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestManagerCreatesAndResolvesProjectRepository(t *testing.T) {
	parent := t.TempDir()
	manager := Repository{}
	projectPath, err := manager.CreateProject(context.Background(), filepath.Join(parent, "new-project"))
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(projectPath, ".git")); err != nil {
		t.Fatalf("expected initialized Git repository: %v", err)
	}
	resolved, err := manager.ResolveProject(context.Background(), projectPath)
	if err != nil {
		t.Fatalf("ResolveProject() error = %v", err)
	}
	if resolved != projectPath {
		t.Fatalf("ResolveProject() = %q, want %q", resolved, projectPath)
	}
	branch, err := manager.DefaultBranch(context.Background(), projectPath)
	if err != nil || branch == "" {
		t.Fatalf("DefaultBranch() = %q, error = %v", branch, err)
	}
	branches, err := manager.ListBranches(context.Background(), projectPath)
	if err != nil || len(branches) == 0 || branches[0] != branch {
		t.Fatalf("ListBranches() = %#v, error = %v", branches, err)
	}
	if exists, err := manager.BranchExists(context.Background(), projectPath, branch); err != nil || exists {
		t.Fatalf("BranchExists(%q) = %v, error = %v; unborn branches are not valid task bases", branch, exists, err)
	}
}

func TestDefaultBranchPrefersOriginHead(t *testing.T) {
	parent := t.TempDir()
	repository := filepath.Join(parent, "repository")
	manager := Repository{}
	path, err := manager.CreateProject(context.Background(), repository)
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	repositoryTestGit(t, path, "config", "user.email", "test@example.com")
	repositoryTestGit(t, path, "config", "user.name", "AI Factory Test")
	if err := os.WriteFile(filepath.Join(path, "README.md"), []byte("test\n"), 0600); err != nil {
		t.Fatal(err)
	}
	repositoryTestGit(t, path, "add", "README.md")
	repositoryTestGit(t, path, "commit", "-m", "initial")
	repositoryTestGit(t, path, "branch", "release/next")
	repositoryTestGit(t, path, "update-ref", "refs/remotes/origin/release/next", "HEAD")
	repositoryTestGit(t, path, "symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/release/next")

	branch, err := manager.DefaultBranch(context.Background(), path)
	if err != nil || branch != "origin/release/next" {
		t.Fatalf("DefaultBranch() = %q, error = %v", branch, err)
	}
}

func repositoryTestGit(t *testing.T, directory string, args ...string) {
	t.Helper()
	if output, err := combinedOutput(t.Context(), directory, args...); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, output)
	}
}

func TestDefaultBranchRejectsNonGitDirectory(t *testing.T) {
	manager := Repository{}
	directory := t.TempDir()
	ceiling := filepath.Dir(directory)
	t.Setenv("GIT_CEILING_DIRECTORIES", ceiling)
	t.Setenv("GIT_DIR", "")
	t.Setenv("GIT_WORK_TREE", "")
	t.Setenv("GIT_COMMON_DIR", "")
	if _, err := manager.ResolveProject(context.Background(), directory); err == nil || strings.TrimSpace(err.Error()) == "" {
		t.Fatal("ResolveProject() succeeded for a non-Git directory")
	}
}

func TestMergeAndPushBranch(t *testing.T) {
	ctx := t.Context()
	repo := Repository{}
	path, err := repo.CreateProject(ctx, filepath.Join(t.TempDir(), "project"))
	if err != nil {
		t.Fatal(err)
	}
	run := func(args ...string) { repositoryTestGit(t, path, args...) }
	run("symbolic-ref", "HEAD", "refs/heads/main")
	run("config", "user.email", "test@example.com")
	run("config", "user.name", "Test")
	run("commit", "--allow-empty", "-m", "base")
	run("checkout", "-b", "feature")
	run("commit", "--allow-empty", "-m", "feature")
	run("checkout", "main")
	if err := repo.MergeBranch(ctx, path, "feature"); err != nil {
		t.Fatal(err)
	}
	remote := t.TempDir()
	run("init", "--bare", remote)
	run("remote", "add", "origin", remote)
	if err := repo.PushBranch(ctx, path, "origin", "main"); err != nil {
		t.Fatal(err)
	}
	got, err := gitOutput(ctx, remote, "rev-parse", "refs/heads/main")
	if err != nil {
		t.Fatal(err)
	}
	want, err := gitOutput(ctx, path, "rev-parse", "refs/heads/feature")
	if err != nil || got != want {
		t.Fatalf("remote main = %q, want %q (%v)", got, want, err)
	}
	if err := repo.PushBranch(ctx, path, "missing", "main"); err == nil {
		t.Fatal("accepted unknown remote")
	}
	if err := repo.MergeBranch(ctx, path, "HEAD~1"); err == nil {
		t.Fatal("accepted revision expression")
	}
}
