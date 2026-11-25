package model

import "time"

const TaskStatusBacklog = "backlog"

// Task keeps the selected base branch stable for its worktree and eventual PR.
type Task struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"project_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	BaseBranch  string    `json:"base_branch"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
