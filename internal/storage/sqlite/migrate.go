package sqlite

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	sqlitedriver "github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

func migrateSchema(path string) error {
	source, err := iofs.New(migrationFiles, "migrations")
	if err != nil {
		return fmt.Errorf("load database migrations: %w", err)
	}
	database, err := sql.Open("sqlite", path)
	if err != nil {
		return fmt.Errorf("open database for migrations: %w", err)
	}
	driver, err := sqlitedriver.WithInstance(database, &sqlitedriver.Config{})
	if err != nil {
		_ = database.Close()
		return fmt.Errorf("prepare SQLite migration driver: %w", err)
	}
	migration, err := migrate.NewWithInstance("iofs", source, "sqlite", driver)
	if err != nil {
		_ = database.Close()
		return fmt.Errorf("prepare database migrations: %w", err)
	}
	return applyMigrations(migration)
}

func applyMigrations(migration *migrate.Migrate) error {
	version, dirty, err := migration.Version()
	if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
		return closeMigration(migration, fmt.Errorf("read database schema version: %w", err))
	}
	if dirty || version > 3 {
		message := fmt.Sprintf("database schema version %d is dirty; repair it before starting AI Factory", version)
		if !dirty {
			message = fmt.Sprintf("database schema version %d is newer than this AI Factory build supports", version)
		}
		return closeMigration(migration, errors.New(message))
	}
	if err := migration.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return closeMigration(migration, fmt.Errorf("apply database migrations: %w", err))
	}
	return closeMigration(migration, nil)
}

func closeMigration(migration *migrate.Migrate, result error) error {
	sourceErr, databaseErr := migration.Close()
	return errors.Join(result, sourceErr, databaseErr)
}
