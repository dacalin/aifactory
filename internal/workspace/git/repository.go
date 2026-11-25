package git

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/dacalin/aifactory/internal/workspace"
)

type Repository struct{}

var _ workspace.Repository = Repository{}

func (Repository) CreateProject(ctx context.Context, requestedPath string) (string, error) {
	path, err := canonicalFuturePath(requestedPath)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(path, 0755); err != nil {
		return "", fmt.Errorf("create project directory: %w", err)
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return "", fmt.Errorf("read project directory: %w", err)
	}
	for _, entry := range entries {
		if entry.Name() != ".git" {
			return "", fmt.Errorf("new project directory must be empty: %s", path)
		}
	}
	if len(entries) == 1 {
		isRepository, err := isEmptyGitRepository(ctx, path)
		if err != nil {
			return "", err
		}
		if !isRepository {
			return "", fmt.Errorf("new project directory must be empty: %s", path)
		}
		return filepath.EvalSymlinks(path)
	}
	if output, err := combinedOutput(ctx, path, "init"); err != nil {
		return "", fmt.Errorf("initialize Git repository: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return filepath.EvalSymlinks(path)
}

func isEmptyGitRepository(ctx context.Context, path string) (bool, error) {
	if output, err := gitOutput(ctx, path, "rev-parse", "--is-inside-work-tree"); err != nil || strings.TrimSpace(output) != "true" {
		return false, nil
	}
	output, err := gitOutput(ctx, path, "rev-list", "--all", "--count")
	if err != nil {
		return false, fmt.Errorf("inspect existing Git repository: %w", err)
	}
	if strings.TrimSpace(output) != "0" {
		return false, nil
	}
	output, err = gitOutput(ctx, path, "status", "--porcelain", "--untracked-files=all")
	if err != nil {
		return false, fmt.Errorf("inspect existing Git worktree: %w", err)
	}
	return strings.TrimSpace(output) == "", nil
}

func (Repository) ResolveProject(ctx context.Context, requestedPath string) (string, error) {
	if strings.TrimSpace(requestedPath) == "" {
		return "", errors.New("project folder is required")
	}
	path, err := filepath.Abs(requestedPath)
	if err != nil {
		return "", fmt.Errorf("resolve project path: %w", err)
	}
	path, err = filepath.EvalSymlinks(filepath.Clean(path))
	if err != nil {
		return "", fmt.Errorf("resolve project folder: %w", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("inspect project folder: %w", err)
	}
	if !info.IsDir() {
		return "", errors.New("selected project path is not a directory")
	}
	output, err := gitOutput(ctx, path, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", errors.New("selected project folder must be inside a Git repository")
	}
	root := strings.TrimSpace(output)
	if root == "" {
		return "", errors.New("could not resolve project Git repository root")
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return "", fmt.Errorf("resolve project repository root: %w", err)
	}
	return root, nil
}

func (Repository) DefaultBranch(ctx context.Context, repository string) (string, error) {
	remoteDefault, _ := gitOutput(ctx, repository, "symbolic-ref", "--quiet", "--short", "refs/remotes/origin/HEAD")
	preferred := []string{strings.TrimSpace(remoteDefault), "main", "master", "origin/main", "origin/master"}
	if branch, found, err := firstExistingBranch(ctx, repository, preferred); err != nil {
		return "", err
	} else if found {
		return branch, nil
	}
	if branch, found, err := currentBranch(ctx, repository); err != nil {
		return "", err
	} else if found {
		return branch, nil
	}
	branches, err := (Repository{}).ListBranches(ctx, repository)
	if err != nil {
		return "", err
	}
	if len(branches) > 0 {
		return branches[0], nil
	}
	return "", errors.New("project repository has no default branch")
}

func firstExistingBranch(ctx context.Context, repository string, candidates []string) (string, bool, error) {
	for _, branch := range candidates {
		if branch == "" {
			continue
		}
		exists, err := (Repository{}).BranchExists(ctx, repository, branch)
		if err != nil {
			return "", false, err
		}
		if exists {
			return branch, true, nil
		}
	}
	return "", false, nil
}

func currentBranch(ctx context.Context, repository string) (string, bool, error) {
	for _, args := range [][]string{
		{"branch", "--show-current"},
		{"symbolic-ref", "--quiet", "--short", "HEAD"},
	} {
		branch, err := gitOutput(ctx, repository, args...)
		branch = strings.TrimSpace(branch)
		if err == nil && branch != "" {
			exists, branchErr := (Repository{}).BranchExists(ctx, repository, branch)
			if branchErr != nil {
				return "", false, branchErr
			}
			if exists {
				return branch, true, nil
			}
		}
	}
	return "", false, nil
}

func (Repository) ListBranches(ctx context.Context, repository string) ([]string, error) {
	output, err := gitOutput(ctx, repository, "for-each-ref", "--format=%(refname:short)", "refs/heads", "refs/remotes")
	if err != nil {
		return nil, fmt.Errorf("list project branches: %w", err)
	}
	branches := make([]string, 0)
	seen := map[string]bool{}
	for _, branch := range strings.Split(output, "\n") {
		branch = strings.TrimSpace(branch)
		if branch == "" || strings.HasSuffix(branch, "/HEAD") || seen[branch] {
			continue
		}
		seen[branch] = true
		branches = append(branches, branch)
	}
	if len(branches) == 0 {
		if branch, err := gitOutput(ctx, repository, "symbolic-ref", "--quiet", "--short", "HEAD"); err == nil {
			branches = append(branches, strings.TrimSpace(branch))
		}
	}
	sort.Strings(branches)
	return branches, nil
}

func (manager Repository) BranchExists(ctx context.Context, repository, branch string) (bool, error) {
	if _, err := gitOutput(ctx, repository, "check-ref-format", "--branch", branch); err != nil {
		return false, fmt.Errorf("invalid project branch %q", branch)
	}
	branches, err := manager.ListBranches(ctx, repository)
	if err != nil {
		return false, err
	}
	for _, candidate := range branches {
		if candidate == branch {
			_, err := gitOutput(ctx, repository, "rev-parse", "--verify", "--quiet", "--end-of-options", branch+"^{commit}")
			if err == nil {
				return true, nil
			}
			var exitError *exec.ExitError
			if errors.As(err, &exitError) && exitError.ExitCode() == 1 {
				return false, nil
			}
			return false, fmt.Errorf("verify project branch %q: %w", branch, err)
		}
	}
	return false, nil
}

func gitOutput(ctx context.Context, repository string, args ...string) (string, error) {
	output, err := outputBytes(ctx, repository, args...)
	if err != nil {
		return "", err
	}
	return string(output), nil
}

func canonicalFuturePath(requestedPath string) (string, error) {
	if strings.TrimSpace(requestedPath) == "" {
		return "", errors.New("project folder is required")
	}
	absolute, err := filepath.Abs(requestedPath)
	if err != nil {
		return "", fmt.Errorf("resolve project path: %w", err)
	}
	absolute = filepath.Clean(absolute)
	if _, err := os.Stat(absolute); err == nil {
		resolved, err := filepath.EvalSymlinks(absolute)
		if err != nil {
			return "", fmt.Errorf("resolve project folder: %w", err)
		}
		return resolved, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("inspect project folder: %w", err)
	}
	parent, leaf := filepath.Dir(absolute), filepath.Base(absolute)
	for {
		if _, err := os.Stat(parent); err == nil {
			break
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("inspect project parent: %w", err)
		}
		next := filepath.Dir(parent)
		if next == parent {
			return "", errors.New("no existing parent directory for project path")
		}
		leaf = filepath.Join(filepath.Base(parent), leaf)
		parent = next
	}
	resolvedParent, err := filepath.EvalSymlinks(parent)
	if err != nil {
		return "", fmt.Errorf("resolve project parent: %w", err)
	}
	return filepath.Join(resolvedParent, leaf), nil
}

func (Repository) MergeBranch(ctx context.Context, worktree, branch string) error {
	ref, err := localBranchRef(ctx, worktree, branch)
	if err != nil {
		return err
	}
	output, err := combinedOutput(ctx, worktree, "merge", "--no-edit", "--", ref)
	if err != nil {
		return fmt.Errorf("merge branch %q: %w: %s", branch, err, strings.TrimSpace(string(output)))
	}
	return nil
}

func (Repository) PushBranch(ctx context.Context, repository, remote, branch string) error {
	ref, err := localBranchRef(ctx, repository, branch)
	if err != nil {
		return err
	}
	remotes, err := gitOutput(ctx, repository, "remote")
	if err != nil {
		return fmt.Errorf("list Git remotes: %w", err)
	}
	if remote == "" || !slices.Contains(strings.Split(strings.TrimSuffix(remotes, "\n"), "\n"), remote) {
		return fmt.Errorf("unknown Git remote %q", remote)
	}
	output, err := combinedOutput(ctx, repository, "push", "--", remote, ref+":"+ref)
	if err != nil {
		return fmt.Errorf("push branch %q to %q: %w: %s", branch, remote, err, strings.TrimSpace(string(output)))
	}
	return nil
}

func localBranchRef(ctx context.Context, repository, branch string) (string, error) {
	ref := "refs/heads/" + branch
	if _, err := gitOutput(ctx, repository, "check-ref-format", ref); err != nil {
		return "", fmt.Errorf("invalid local branch %q: %w", branch, err)
	}
	if _, err := gitOutput(ctx, repository, "show-ref", "--verify", "--quiet", ref); err != nil {
		return "", fmt.Errorf("resolve local branch %q: %w", branch, err)
	}
	return ref, nil
}
