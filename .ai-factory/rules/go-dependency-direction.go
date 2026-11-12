package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
)

// Each source/target pair covers descendant packages.
var forbiddenDependencies = []struct{ From, To string }{
	{"internal/agents", "internal/llm"},
	{"internal/llm", "internal/commands"},
}

func init() { register("go-dependency-direction", checkDependencies) }

func checkDependencies(repo *repository) error {
	cmd := exec.Command("go", "list", "-json", "./...")
	cmd.Dir = repo.root
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("resolve direct imports: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(output))
	var issues []error
	for {
		var pkg struct {
			ImportPath string
			Imports    []string
		}
		err := decoder.Decode(&pkg)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		for _, dependency := range pkg.Imports {
			for _, boundary := range forbiddenDependencies {
				if within(pkg.ImportPath, repo.packagePath(boundary.From)) && within(dependency, repo.packagePath(boundary.To)) {
					issues = append(issues, fmt.Errorf("%s must not import %s", pkg.ImportPath, dependency))
				}
			}
		}
	}
	return errors.Join(issues...)
}
