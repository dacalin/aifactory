# Keep direct database access inside persistence

## Scope

This rule applies to Go source, including tests, beneath the configured source
roots. Generated files, vendored source, hidden metadata paths, generated rule
artifacts, and configured exclusions are not scanned. It recognizes direct
imports of the configured database client, driver, and migration package roots,
as well as imports of configured concrete persistence adapters.

## Requirement

Persistence is an architectural role, not merely a folder name. Direct database
access belongs only to the configured persistence root and its descendants.
Code outside that boundary may not import a database client, driver, or migration
package.

The public persistence boundary owns the configured lifecycle interface with
`Close() error`. Each configured concrete adapter declares its conformance to
that interface in the file that declares its struct. Only configured composition
files may import a concrete adapter and construct it; the public contract package
does not expose an adapter-selecting `Open` wrapper.

Add every new database client, driver, migration library, and concrete adapter
root to the checker settings before using it. Those editable settings make the
set of direct database APIs measurable.

## Valid example

```go
// internal/storage/database.go
package storage

type Database interface {
	Close() error
}
```

```go
// internal/storage/sqlite/db.go
package sqlite

import "example.com/project/internal/storage"

type DB struct{}

var _ storage.Database = (*DB)(nil)

func Open(path string) (*DB, error) {
	return &DB{}, nil
}
```

```go
// bootstrap.go
package main

import "example.com/project/internal/storage/sqlite"

func run(path string) error {
	database, err := sqlite.Open(path)
	if err != nil {
		return err
	}
	defer database.Close()
	return nil
}
```

## Run the check

From the project root, run:

```sh
./.ai-factory/rules/checkrules database-access-storage
```

Configure the persistence root, source roots, exclusions, recognized database
packages, concrete adapter roots, composition files, adapter contract structs,
and lifecycle contract name in the settings block at the top of
`database-access-storage.go`.
