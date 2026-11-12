package main

import (
	"errors"
	"fmt"
	"go/ast"
	"strings"
)

// Interface naming paths, exclusions, and prohibited fragments are configurable.
var interfaceNameSettings = struct {
	Roots              []string
	ExcludedRoots      []string
	ForbiddenFragments []string
}{
	Roots:              []string{"."},
	ExcludedRoots:      []string{},
	ForbiddenFragments: []string{"interface"},
}

func init() { register("interface-type-names", checkInterfaceTypeNames) }

func checkInterfaceTypeNames(repo *repository) error {
	files, err := repo.sources(true)
	if err != nil {
		return err
	}
	var issues []error
	for _, file := range files {
		if !matchesInterfaceNameScope(file.dir) {
			continue
		}
		ast.Inspect(file.tree, func(node ast.Node) bool {
			spec, ok := node.(*ast.TypeSpec)
			if !ok {
				return true
			}
			if _, ok := spec.Type.(*ast.InterfaceType); !ok {
				return true
			}
			if fragment := forbiddenInterfaceNameFragment(spec.Name.Name); fragment != "" {
				issues = append(issues, file.at(spec.Name.Pos(), fmt.Sprintf("interface %s must not include %q in its name", spec.Name.Name, fragment)))
			}
			return true
		})
	}
	return errors.Join(issues...)
}

func matchesInterfaceNameScope(directory string) bool {
	if !matchesAnyRoot(directory, interfaceNameSettings.Roots) {
		return false
	}
	return !matchesAnyRoot(directory, interfaceNameSettings.ExcludedRoots)
}

func matchesAnyRoot(directory string, roots []string) bool {
	for _, root := range roots {
		if within(directory, root) {
			return true
		}
	}
	return false
}

func forbiddenInterfaceNameFragment(name string) string {
	for _, fragment := range interfaceNameSettings.ForbiddenFragments {
		if strings.Contains(strings.ToLower(name), strings.ToLower(fragment)) {
			return fragment
		}
	}
	return ""
}
