package git

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dacalin/aifactory/internal/model"
	"github.com/dacalin/aifactory/internal/workspace"
)

type Manager struct {
	workspacesRoot string
}

var _ workspace.Manager = (*Manager)(nil)

func NewManager(workspacesRoot string) *Manager {
	return &Manager{workspacesRoot: workspacesRoot}
}

func (manager *Manager) Create(ctx context.Context, project model.Project, task model.Task) (workspace.Workspace, error) {
	if err := validateSegment("project ID", project.ID); err != nil {
		return workspace.Workspace{}, err
	}
	if err := validateSegment("task ID", task.ID); err != nil {
		return workspace.Workspace{}, err
	}
	if task.ProjectID != project.ID {
		return workspace.Workspace{}, errors.New("task does not belong to the supplied project")
	}
	baseBranch := strings.TrimSpace(task.BaseBranch)
	if baseBranch == "" {
		baseBranch = project.BaseBranch
	}
	if baseBranch == "" {
		return workspace.Workspace{}, errors.New("task base branch is not configured")
	}
	projectPath, err := resolveRepositoryRoot(ctx, project.Path)
	if err != nil {
		return workspace.Workspace{}, fmt.Errorf("resolve project repository: %w", err)
	}
	root, err := filepath.Abs(manager.workspacesRoot)
	if err != nil {
		return workspace.Workspace{}, fmt.Errorf("resolve task workspace directory: %w", err)
	}
	if strings.TrimSpace(manager.workspacesRoot) == "" {
		return workspace.Workspace{}, errors.New("task workspace directory is not configured")
	}
	path := filepath.Join(root, project.ID, task.ID)
	branch := "aifactory/" + project.ID + "/" + task.ID
	if existing, err := manager.existingWorkspace(ctx, project, task, projectPath, path, branch, baseBranch); err != nil {
		return workspace.Workspace{}, err
	} else if existing != nil {
		return *existing, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return workspace.Workspace{}, fmt.Errorf("create task workspace parent: %w", err)
	}
	if output, err := combinedOutput(ctx, projectPath, "worktree", "add", "-b", branch, path, baseBranch); err != nil {
		return workspace.Workspace{}, fmt.Errorf("create Git worktree: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return workspace.Workspace{
		ProjectID: project.ID, TaskID: task.ID, ProjectPath: projectPath, Path: path, Branch: branch, BaseBranch: baseBranch,
	}, nil
}

func (manager *Manager) existingWorkspace(ctx context.Context, project model.Project, task model.Task, projectPath, path, branch, baseBranch string) (*workspace.Workspace, error) {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("inspect task workspace: %w", err)
	}
	root, err := resolveRepositoryRoot(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("task workspace path already exists but is not a Git worktree: %s", path)
	}
	resolvedPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return nil, fmt.Errorf("resolve task workspace path: %w", err)
	}
	if root != resolvedPath {
		return nil, fmt.Errorf("task workspace path already belongs to another Git worktree: %s", path)
	}
	projectGitDir, err := repositoryCommonDir(ctx, projectPath)
	if err != nil {
		return nil, fmt.Errorf("inspect project repository: %w", err)
	}
	workspaceGitDir, err := repositoryCommonDir(ctx, path)
	if err != nil || projectGitDir != workspaceGitDir {
		return nil, fmt.Errorf("task workspace path belongs to a different project repository: %s", path)
	}
	output, err := outputBytes(ctx, path, "branch", "--show-current")
	if err != nil {
		return nil, fmt.Errorf("inspect task workspace branch: %w", err)
	}
	branch = strings.TrimSpace(string(output))
	return &workspace.Workspace{
		ProjectID: project.ID, TaskID: task.ID, ProjectPath: projectPath, Path: path, Branch: branch, BaseBranch: baseBranch,
	}, nil
}

func (manager *Manager) Remove(ctx context.Context, target workspace.Workspace) error {
	root, err := filepath.Abs(manager.workspacesRoot)
	if err != nil {
		return fmt.Errorf("resolve task workspace directory: %w", err)
	}
	path, err := filepath.Abs(target.Path)
	if err != nil {
		return fmt.Errorf("resolve task workspace path: %w", err)
	}
	relative, err := filepath.Rel(root, path)
	if err != nil || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return errors.New("task workspace is outside the managed workspace directory")
	}
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return fmt.Errorf("resolve managed workspace directory: %w", err)
	}
	resolvedPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return fmt.Errorf("resolve task workspace path: %w", err)
	}
	relative, err = filepath.Rel(resolvedRoot, resolvedPath)
	if err != nil || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return errors.New("task workspace resolves outside the managed workspace directory")
	}
	projectPath, err := resolveRepositoryRoot(ctx, target.ProjectPath)
	if err != nil {
		return fmt.Errorf("resolve project repository: %w", err)
	}
	if output, err := combinedOutput(ctx, projectPath, "worktree", "remove", "--force", path); err != nil {
		return fmt.Errorf("remove Git worktree: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func resolveRepositoryRoot(ctx context.Context, path string) (string, error) {
	output, err := outputBytes(ctx, path, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}
	root, err := filepath.Abs(strings.TrimSpace(string(output)))
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(root)
}

func repositoryCommonDir(ctx context.Context, path string) (string, error) {
	output, err := outputBytes(ctx, path, "rev-parse", "--git-common-dir")
	if err != nil {
		return "", err
	}
	commonDir := strings.TrimSpace(string(output))
	if !filepath.IsAbs(commonDir) {
		commonDir = filepath.Join(path, commonDir)
	}
	commonDir, err = filepath.Abs(commonDir)
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(commonDir)
}

func validateSegment(label, value string) error {
	if value == "" || value == "." || value == ".." || filepath.Base(value) != value || strings.ContainsAny(value, "/\\\x00") {
		return fmt.Errorf("%s must be a safe single path segment", label)
	}
	return nil
}
