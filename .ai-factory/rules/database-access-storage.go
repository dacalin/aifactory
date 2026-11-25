package main

import (
	"errors"
	"fmt"
	"go/ast"
	"strconv"
	"strings"
)

// Persistence boundaries and recognized direct database dependencies are editable here.
type databaseAccessConfiguration struct {
	StorageRoot         string
	SourceRoots         []string
	ExcludedRoots       []string
	DatabaseImportRoots []string
	AdapterRoots        []string
	CompositionFiles    []string
	LifecycleInterface  string
	AdapterContracts    []databaseAdapterContract
}

type databaseAdapterContract struct {
	Root, Struct string
}

var databaseAccessSettings = databaseAccessConfiguration{
	StorageRoot:   "internal/storage",
	SourceRoots:   []string{"."},
	ExcludedRoots: []string{},
	DatabaseImportRoots: []string{
		"database/sql",
		"modernc.org/sqlite",
		"github.com/golang-migrate/migrate/v4",
	},
	AdapterRoots:       []string{"internal/storage/sqlite"},
	CompositionFiles:   []string{"bootstrap.go"},
	LifecycleInterface: "Database",
	AdapterContracts: []databaseAdapterContract{
		{Root: "internal/storage/sqlite", Struct: "DB"},
	},
}

func init() { register("database-access-storage", checkDatabaseAccess) }

func checkDatabaseAccess(repo *repository) error {
	files, err := repo.sources(true)
	if err != nil {
		return err
	}
	var issues []error
	for _, file := range files {
		if !databaseAccessInScope(file.dir) {
			continue
		}
		for _, imported := range directImportPaths(file) {
			if root := databaseImportRoot(imported); root != "" && !within(file.dir, databaseAccessSettings.StorageRoot) {
				issues = append(issues, fmt.Errorf("%s: direct database package %s must be imported only under %s", file.name, root, databaseAccessSettings.StorageRoot))
			}
			if root := adapterImportRoot(repo, imported); root != "" && !databaseAdapterImportAllowed(file.name) {
				issues = append(issues, fmt.Errorf("%s: concrete storage adapter %s must be imported only by configured composition files", file.name, root))
			}
		}
	}
	issues = append(issues, checkDatabaseLifecycle(repo))
	return errors.Join(issues...)
}

func directImportPaths(file *sourceFile) []string {
	paths := make([]string, 0, len(file.tree.Imports))
	for _, specification := range file.tree.Imports {
		imported, err := strconv.Unquote(specification.Path.Value)
		if err == nil {
			paths = append(paths, imported)
		}
	}
	return paths
}

func databaseAccessInScope(directory string) bool {
	return matchesAnyRoot(directory, databaseAccessSettings.SourceRoots) && !matchesAnyRoot(directory, databaseAccessSettings.ExcludedRoots)
}

func databaseAdapterImportAllowed(name string) bool {
	for _, composition := range databaseAccessSettings.CompositionFiles {
		if name == composition {
			return true
		}
	}
	return false
}

func databaseImportRoot(imported string) string {
	for _, root := range databaseAccessSettings.DatabaseImportRoots {
		if matchesImportRoot(imported, root) {
			return root
		}
	}
	return ""
}

func adapterImportRoot(repo *repository, imported string) string {
	for _, root := range databaseAccessSettings.AdapterRoots {
		if matchesImportRoot(imported, repo.packagePath(root)) {
			return root
		}
	}
	return ""
}

func matchesImportRoot(imported, root string) bool {
	return imported == root || strings.HasPrefix(imported, root+"/")
}

func checkDatabaseLifecycle(repo *repository) error {
	var issues []error
	definitions := repo.types[databaseAccessSettings.StorageRoot]
	definition, exists := definitions[databaseAccessSettings.LifecycleInterface]
	if !exists {
		issues = append(issues, fmt.Errorf("%s must declare interface %s", databaseAccessSettings.StorageRoot, databaseAccessSettings.LifecycleInterface))
	} else if contract, ok := definition.spec.Type.(*ast.InterfaceType); !ok {
		issues = append(issues, definition.source.at(definition.spec.Pos(), fmt.Sprintf("%s must be an interface", databaseAccessSettings.LifecycleInterface)))
	} else if !databaseCloseMethod(contract) {
		issues = append(issues, definition.source.at(definition.spec.Pos(), fmt.Sprintf("%s must declare Close() error", databaseAccessSettings.LifecycleInterface)))
	}
	if !exists {
		return errors.Join(issues...)
	}
	target := repo.packagePath(databaseAccessSettings.StorageRoot) + "." + databaseAccessSettings.LifecycleInterface
	for _, adapter := range databaseAccessSettings.AdapterContracts {
		definition, found := repo.types[adapter.Root][adapter.Struct]
		if !found {
			issues = append(issues, fmt.Errorf("%s must declare struct %s", adapter.Root, adapter.Struct))
			continue
		}
		if _, ok := definition.spec.Type.(*ast.StructType); !ok {
			issues = append(issues, definition.source.at(definition.spec.Pos(), fmt.Sprintf("%s must be a struct", adapter.Struct)))
			continue
		}
		if !repo.hasAssertionInFile(definition.source, adapter.Struct, target) {
			issues = append(issues, definition.source.at(definition.spec.Pos(), fmt.Sprintf("%s must assert %s in its declaration file", adapter.Struct, target)))
		}
	}
	return errors.Join(issues...)
}

func databaseCloseMethod(contract *ast.InterfaceType) bool {
	for _, method := range contract.Methods.List {
		if len(method.Names) != 1 || method.Names[0].Name != "Close" {
			continue
		}
		signature, ok := method.Type.(*ast.FuncType)
		if !ok || signature.Params != nil && len(signature.Params.List) != 0 || signature.Results == nil || len(signature.Results.List) != 1 {
			continue
		}
		if result, ok := unwrap(signature.Results.List[0].Type).(*ast.Ident); ok && result.Name == "error" {
			return true
		}
	}
	return false
}
