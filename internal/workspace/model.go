package workspace

import "path/filepath"

// Workspace is the isolated Git working tree assigned to a task.
type Workspace struct {
	ProjectID   string `json:"project_id"`
	TaskID      string `json:"task_id"`
	ProjectPath string `json:"project_path"`
	Path        string `json:"path"`
	Branch      string `json:"branch"`
	BaseBranch  string `json:"base_branch"`
}

func (workspace Workspace) WorkingDirectory() string {
	return filepath.Clean(workspace.Path)
}
