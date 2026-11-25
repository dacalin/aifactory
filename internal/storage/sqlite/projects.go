package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	projectdomain "github.com/dacalin/aifactory/internal/app/project"
	"github.com/dacalin/aifactory/internal/model"
)

type ProjectRepository struct {
	database *sql.DB
}

func NewProjectRepository(database *sql.DB) *ProjectRepository {
	return &ProjectRepository{database: database}
}

func (store *ProjectRepository) CreateOrGetByPath(ctx context.Context, project model.Project) (model.Project, bool, error) {
	result, err := store.database.ExecContext(ctx,
		`INSERT INTO projects (id, name, path, base_branch, created_at) VALUES (?, ?, ?, ?, ?) ON CONFLICT(path) DO NOTHING`,
		project.ID, project.Name, project.Path, project.BaseBranch, project.CreatedAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return model.Project{}, false, fmt.Errorf("save project: %w", err)
	}
	inserted, err := result.RowsAffected()
	if err != nil {
		return model.Project{}, false, fmt.Errorf("check project registration: %w", err)
	}
	if inserted == 1 {
		return project, true, nil
	}
	_, err = store.database.ExecContext(ctx,
		`UPDATE projects SET base_branch = ? WHERE path = ? AND base_branch = ''`, project.BaseBranch, project.Path)
	if err != nil {
		return model.Project{}, false, fmt.Errorf("set project base branch: %w", err)
	}
	existing, err := scanProject(store.database.QueryRowContext(ctx,
		`SELECT id, name, path, base_branch, created_at FROM projects WHERE path = ?`, project.Path))
	return existing, false, err
}

func (store *ProjectRepository) Get(ctx context.Context, id string) (model.Project, error) {
	project, err := scanProject(store.database.QueryRowContext(ctx,
		`SELECT id, name, path, base_branch, created_at FROM projects WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return model.Project{}, projectdomain.ErrProjectNotFound
	}
	return project, err
}

func (store *ProjectRepository) List(ctx context.Context) ([]model.Project, error) {
	rows, err := store.database.QueryContext(ctx,
		`SELECT id, name, path, base_branch, created_at FROM projects ORDER BY name COLLATE NOCASE, created_at`)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	defer rows.Close()
	projects := make([]model.Project, 0)
	for rows.Next() {
		project, err := scanProject(rows)
		if err != nil {
			return nil, fmt.Errorf("read project: %w", err)
		}
		projects = append(projects, project)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate projects: %w", err)
	}
	return projects, nil
}

func (store *ProjectRepository) UpdateBaseBranch(ctx context.Context, id, branch string) (model.Project, error) {
	result, err := store.database.ExecContext(ctx, `UPDATE projects SET base_branch = ? WHERE id = ?`, branch, id)
	if err != nil {
		return model.Project{}, fmt.Errorf("update project base branch: %w", err)
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return model.Project{}, fmt.Errorf("check project base branch update: %w", err)
	}
	if updated == 0 {
		return model.Project{}, projectdomain.ErrProjectNotFound
	}
	return store.Get(ctx, id)
}

func scanProject(row scanner) (model.Project, error) {
	var project model.Project
	var createdAt string
	if err := row.Scan(&project.ID, &project.Name, &project.Path, &project.BaseBranch, &createdAt); err != nil {
		return model.Project{}, err
	}
	parsed, err := time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return model.Project{}, fmt.Errorf("parse project creation time: %w", err)
	}
	project.CreatedAt = parsed.UTC()
	return project, nil
}
