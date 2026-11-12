package main

import (
	"errors"
	"fmt"
)

// Concrete infrastructure packages can only be imported by this composition root.
var bootstrapSettings = struct {
	File, Function      string
	ImplementationRoots []string
}{
	"bootstrap.go", "bootstrap", []string{"internal/llm"},
}

func init() { register("dependency-injection-bootstrap", checkBootstrap) }

func checkBootstrap(repo *repository) error {
	files, err := repo.sources(false)
	if err != nil {
		return err
	}
	var issues []error
	found := false
	for _, file := range files {
		if file.name == bootstrapSettings.File {
			found = true
			if file.tree.Name.Name != "main" || namedFunction(file, bootstrapSettings.Function) == nil {
				issues = append(issues, fmt.Errorf("%s must declare package main and %s", file.name, bootstrapSettings.Function))
			}
			continue
		}
		for _, imported := range file.imports {
			for _, root := range bootstrapSettings.ImplementationRoots {
				if within(imported, repo.packagePath(root)) {
					issues = append(issues, fmt.Errorf("%s: concrete implementation %s must be wired in %s", file.name, imported, bootstrapSettings.File))
				}
			}
		}
	}
	if !found {
		issues = append(issues, fmt.Errorf("missing composition root %s", bootstrapSettings.File))
	}
	return errors.Join(issues...)
}
