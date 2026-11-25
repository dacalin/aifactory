package main

import (
	"errors"
	"fmt"
	"go/ast"
	"path"
)

// Ownership roots and declarations can be adapted to another layout.
type packageOwnershipSettings struct {
	Agents, Options, SharedCLI string
	Option, Runner             string
	Providers                  []string
	Retired                    []string
	Contracts                  []string
}

var packageOwnership = packageOwnershipSettings{
	Agents:    "internal/agents",
	Options:   "internal/agents/llm",
	SharedCLI: "internal/agents/llm/runnercli",
	Option:    "Option",
	Runner:    "AgentRunner",
	Providers: []string{
		"internal/agents/llm/codex",
		"internal/agents/llm/claude",
	},
	Retired:   []string{"internal/ports", "internal/adapters"},
	Contracts: []string{"Agent", "AgentRunner"},
}

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
	}
	for _, root := range packageRoots() {
		issues = append(issues, checkPackageRoot(files, root))
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
	if _, exists := repo.types[packageOwnership.Options][packageOwnership.Option]; !exists {
		issues = append(issues, fmt.Errorf("%s must own provider option %s", packageOwnership.Options, packageOwnership.Option))
	}
	for _, root := range packageOwnership.Providers {
		if !providerImplementsRunner(repo, root) {
			issues = append(issues, fmt.Errorf("%s must declare a struct that asserts %s.%s", root, path.Base(packageOwnership.Agents), packageOwnership.Runner))
		}
	}
	return errors.Join(issues...)
}

func packageRoots() []string {
	roots := []string{packageOwnership.Agents, packageOwnership.Options, packageOwnership.SharedCLI}
	return append(roots, packageOwnership.Providers...)
}

func checkPackageRoot(files []*sourceFile, root string) error {
	found := false
	var issues []error
	for _, file := range files {
		if file.dir != root {
			continue
		}
		found = true
		if file.tree.Name.Name != path.Base(root) {
			issues = append(issues, fmt.Errorf("%s: expected package %s", file.name, path.Base(root)))
		}
	}
	if !found {
		issues = append(issues, fmt.Errorf("%s must contain Go source", root))
	}
	return errors.Join(issues...)
}

func providerImplementsRunner(repo *repository, root string) bool {
	target := repo.packagePath(packageOwnership.Agents) + "." + packageOwnership.Runner
	for name, definition := range repo.types[root] {
		if _, ok := definition.spec.Type.(*ast.StructType); ok && repo.hasAssertion(root, name, target) {
			return true
		}
	}
	return false
}
