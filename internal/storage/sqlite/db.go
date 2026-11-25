package sqlite

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/dacalin/aifactory/internal/app/project"
	"github.com/dacalin/aifactory/internal/app/task"
	"github.com/dacalin/aifactory/internal/storage"
	_ "modernc.org/sqlite"
)

type DB struct {
	*sql.DB
}

var _ storage.Database = (*DB)(nil)

func Open(path string) (*DB, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve database path: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, fmt.Errorf("create application data directory: %w", err)
	}
	if err := secureDatabaseFile(path); err != nil {
		return nil, err
	}
	if err := migrateSchema(path); err != nil {
		return nil, err
	}

	database, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open application database: %w", err)
	}
	database.SetMaxOpenConns(1)
	if err := configureDatabase(database); err != nil {
		_ = database.Close()
		return nil, err
	}
	return &DB{DB: database}, nil
}

func secureDatabaseFile(path string) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return fmt.Errorf("create application database file: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close application database file: %w", err)
	}
	if err := os.Chmod(path, 0600); err != nil {
		return fmt.Errorf("secure application database file: %w", err)
	}
	return nil
}

func configureDatabase(database *sql.DB) error {
	if _, err := database.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		return fmt.Errorf("enable SQLite foreign keys: %w", err)
	}
	if _, err := database.Exec(`PRAGMA busy_timeout = 5000`); err != nil {
		return fmt.Errorf("configure application database timeout: %w", err)
	}
	if _, err := database.Exec(`PRAGMA journal_mode = WAL`); err != nil {
		return fmt.Errorf("enable application database WAL mode: %w", err)
	}
	return nil
}

func (database *DB) Close() error {
	return database.DB.Close()
}

func (database *DB) Projects() project.Repository {
	return NewProjectRepository(database.DB)
}

func (database *DB) Tasks() task.Repository {
	return NewTaskRepository(database.DB)
}
