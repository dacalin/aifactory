package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/dacalin/aifactory/internal/app/task"
	"github.com/dacalin/aifactory/internal/model"
)

type TaskRepository struct {
	database *sql.DB
}

func NewTaskRepository(database *sql.DB) *TaskRepository {
	return &TaskRepository{database: database}
}

var _ task.Repository = (*TaskRepository)(nil)

func (repository *TaskRepository) Create(ctx context.Context, task model.Task) (model.Task, error) {
	now := task.CreatedAt.UTC().Format(time.RFC3339Nano)
	_, err := repository.database.ExecContext(ctx,
		`INSERT INTO tasks (id, project_id, title, description, base_branch, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		task.ID, task.ProjectID, task.Title, task.Description, task.BaseBranch, task.Status, now, task.UpdatedAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return model.Task{}, fmt.Errorf("save task: %w", err)
	}
	return task, nil
}

func (repository *TaskRepository) ListByProject(ctx context.Context, projectID string) ([]model.Task, error) {
	rows, err := repository.database.QueryContext(ctx,
		`SELECT id, project_id, title, description, base_branch, status, created_at, updated_at FROM tasks WHERE project_id = ? ORDER BY created_at, id`, projectID)
	if err != nil {
		return nil, fmt.Errorf("list project tasks: %w", err)
	}
	defer rows.Close()
	tasks := make([]model.Task, 0)
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, fmt.Errorf("read task: %w", err)
		}
		tasks = append(tasks, task)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate project tasks: %w", err)
	}
	return tasks, nil
}

func scanTask(row scanner) (model.Task, error) {
	var task model.Task
	var createdAt, updatedAt string
	if err := row.Scan(&task.ID, &task.ProjectID, &task.Title, &task.Description, &task.BaseBranch, &task.Status, &createdAt, &updatedAt); err != nil {
		return model.Task{}, err
	}
	var err error
	if task.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt); err != nil {
		return model.Task{}, fmt.Errorf("parse task creation time: %w", err)
	}
	if task.UpdatedAt, err = time.Parse(time.RFC3339Nano, updatedAt); err != nil {
		return model.Task{}, fmt.Errorf("parse task update time: %w", err)
	}
	task.CreatedAt = task.CreatedAt.UTC()
	task.UpdatedAt = task.UpdatedAt.UTC()
	return task, nil
}
