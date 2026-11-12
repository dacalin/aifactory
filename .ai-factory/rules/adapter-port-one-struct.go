package main

import (
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"path"
	"strings"
	"unicode"
)

// Implementation files own one struct; contract files own one interface.
// Helper-only files are allowed.
var declarationScopes = []struct{ Root, Kind string }{
	{"internal/llm", "struct"},
	{"internal/agents", "interface"},
}

func init() { register("adapter-port-one-struct", checkFocusedDeclarations) }

func checkFocusedDeclarations(repo *repository) error {
	files, err := repo.sources(false)
	if err != nil {
		return err
	}
	var issues []error
	for _, file := range files {
		for _, scope := range declarationScopes {
			if !within(file.dir, scope.Root) {
				continue
			}
			names := declaredNames(file.tree, scope.Kind)
			if len(names) > 1 {
				issues = append(issues, fmt.Errorf("%s: keep at most one %s declaration per file", file.name, scope.Kind))
			}
			if len(names) == 1 && path.Base(file.name) != snakeCase(names[0])+".go" {
				issues = append(issues, fmt.Errorf("%s: %s belongs in %s.go", file.name, names[0], snakeCase(names[0])))
			}
		}
	}
	return errors.Join(issues...)
}

func declaredNames(file *ast.File, kind string) []string {
	var names []string
	for _, declaration := range file.Decls {
		gen, ok := declaration.(*ast.GenDecl)
		if !ok || gen.Tok != token.TYPE {
			continue
		}
		for _, specification := range gen.Specs {
			spec := specification.(*ast.TypeSpec)
			_, structure := spec.Type.(*ast.StructType)
			_, contract := spec.Type.(*ast.InterfaceType)
			if kind == "struct" && structure || kind == "interface" && contract {
				names = append(names, spec.Name.Name)
			}
		}
	}
	return names
}

func snakeCase(identifier string) string {
	runes := []rune(identifier)
	var result strings.Builder
	for index, current := range runes {
		if unicode.IsUpper(current) && index > 0 {
			previous := runes[index-1]
			nextIsLower := index+1 < len(runes) && unicode.IsLower(runes[index+1])
			if unicode.IsLower(previous) || unicode.IsDigit(previous) || unicode.IsUpper(previous) && nextIsLower {
				result.WriteByte('_')
			}
		}
		result.WriteRune(unicode.ToLower(current))
	}
	return result.String()
}
