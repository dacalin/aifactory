package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
)

// Targets always cover descendant packages. Sources can be exact package roots
// or whole package trees so provider helpers can remain usable by providers.
type dependencyBoundary struct {
	From, To        string
	FromDescendants bool
}

var forbiddenDependencies = []dependencyBoundary{
	{"internal/agents", "internal/agents/llm", false},
	{"internal/agents/llm", "internal/commands", true},
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
				if dependencySourceMatches(repo, pkg.ImportPath, boundary) && within(dependency, repo.packagePath(boundary.To)) {
					issues = append(issues, fmt.Errorf("%s must not import %s", pkg.ImportPath, dependency))
				}
			}
		}
	}
	return errors.Join(issues...)
}

func dependencySourceMatches(repo *repository, source string, boundary dependencyBoundary) bool {
	root := repo.packagePath(boundary.From)
	if boundary.FromDescendants {
		return within(source, root)
	}
	return source == root
}
