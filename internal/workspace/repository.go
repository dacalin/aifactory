package workspace

import "context"

// SourceControl manages the Git repository selected or created for a project.
type SourceControl interface {
	CreateProject(context.Context, string) (string, error)
	ResolveProject(context.Context, string) (string, error)
	DefaultBranch(context.Context, string) (string, error)
	ListBranches(context.Context, string) ([]string, error)
	BranchExists(context.Context, string, string) (bool, error)
}

// Repository provides the Git operations available to application services.
type Repository interface {
	SourceControl
	// MergeBranch merges a local branch into the currently checked-out branch.
	MergeBranch(ctx context.Context, worktree, branch string) error
	// PushBranch pushes a local branch to the same branch on a configured remote.
	PushBranch(ctx context.Context, repository, remote, branch string) error
}
