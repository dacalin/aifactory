package main

import (
	"errors"
	"fmt"
	"go/ast"
	"path"
)

// Ownership roots and declarations can be adapted to another layout.
var packageOwnership = struct {
	Agents, Models string
	Retired        []string
	Contracts      []string
}{"internal/agents", "internal/llm", []string{"internal/ports", "internal/adapters"}, []string{"Agent", "AgentRunner"}}

func init() { register("package-organization", checkPackageOrganization) }

func checkPackageOrganization(repo *repository) error {
	files, err := repo.sources(false)
	if err != nil {
		return err
	}
	var issues []error
	for _, file := range files {
		for _, root := range packageOwnership.Retired {
			if within(file.dir, root) {
				issues = append(issues, fmt.Errorf("%s: move retired package source to its responsibility package", file.name))
			}
		}
		for _, root := range []string{packageOwnership.Agents, packageOwnership.Models} {
			if file.dir == root && file.tree.Name.Name != path.Base(root) {
				issues = append(issues, fmt.Errorf("%s: expected package %s", file.name, path.Base(root)))
			}
		}
	}
	definitions := repo.types[packageOwnership.Agents]
	for _, name := range packageOwnership.Contracts {
		definition, exists := definitions[name]
		if !exists {
			issues = append(issues, fmt.Errorf("%s must own interface %s", packageOwnership.Agents, name))
			continue
		}
		if _, ok := definition.spec.Type.(*ast.InterfaceType); !ok {
			issues = append(issues, fmt.Errorf("%s must be an interface owned by agents", name))
		}
	}
	if definition, exists := definitions["RuleAgent"]; !exists {
		issues = append(issues, fmt.Errorf("%s must own RuleAgent", packageOwnership.Agents))
	} else if _, ok := definition.spec.Type.(*ast.StructType); !ok {
		issues = append(issues, fmt.Errorf("RuleAgent must be a struct"))
	}
	if len(repo.types[packageOwnership.Models]) == 0 {
		issues = append(issues, fmt.Errorf("%s must own model implementations", packageOwnership.Models))
	}
	return errors.Join(issues...)
}
