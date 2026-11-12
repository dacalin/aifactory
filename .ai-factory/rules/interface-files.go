package main

import (
	"errors"
	"fmt"
	"go/ast"
	"go/token"
)

func init() { register("interface-files", checkInterfaceFiles) }

// checkInterfaceFiles keeps interface contracts separate from implementations
// and unrelated declarations. Supporting named types are permitted only when an
// interface reaches them through its type expression, directly or transitively.
func checkInterfaceFiles(repo *repository) error {
	files, err := repo.sources(true)
	if err != nil {
		return err
	}

	var issues []error
	for _, file := range files {
		types, interfaces := fileTypes(file.tree)
		if len(interfaces) == 0 {
			continue
		}
		supporting := requiredInterfaceTypes(types, interfaces)
		for _, declaration := range file.tree.Decls {
			switch declaration := declaration.(type) {
			case *ast.FuncDecl:
				issues = append(issues, file.at(declaration.Pos(), "interface-declaring files must not declare functions or methods"))
			case *ast.GenDecl:
				switch declaration.Tok {
				case token.IMPORT:
					continue
				case token.TYPE:
					for _, specification := range declaration.Specs {
						spec := specification.(*ast.TypeSpec)
						if containsStruct(spec.Type) {
							issues = append(issues, file.at(spec.Pos(), fmt.Sprintf("interface-declaring files must not contain struct types in %s", spec.Name.Name)))
							continue
						}
						if _, isInterface := spec.Type.(*ast.InterfaceType); isInterface {
							continue
						}
						if !supporting[spec.Name.Name] {
							issues = append(issues, file.at(spec.Pos(), fmt.Sprintf("supporting type %s is not used by an interface", spec.Name.Name)))
						}
					}
				default:
					issues = append(issues, file.at(declaration.Pos(), fmt.Sprintf("interface-declaring files must not declare %s", declaration.Tok.String())))
				}
			}
		}
	}
	return errors.Join(issues...)
}

func fileTypes(file *ast.File) (map[string]*ast.TypeSpec, []*ast.TypeSpec) {
	types := map[string]*ast.TypeSpec{}
	var interfaces []*ast.TypeSpec
	for _, declaration := range file.Decls {
		gen, ok := declaration.(*ast.GenDecl)
		if !ok || gen.Tok != token.TYPE {
			continue
		}
		for _, specification := range gen.Specs {
			spec := specification.(*ast.TypeSpec)
			types[spec.Name.Name] = spec
			if _, ok := spec.Type.(*ast.InterfaceType); ok {
				interfaces = append(interfaces, spec)
			}
		}
	}
	return types, interfaces
}

func requiredInterfaceTypes(types map[string]*ast.TypeSpec, interfaces []*ast.TypeSpec) map[string]bool {
	required := map[string]bool{}
	var visit func(ast.Node)
	visit = func(node ast.Node) {
		ast.Inspect(node, func(node ast.Node) bool {
			switch expression := node.(type) {
			case *ast.Field:
				visit(expression.Type)
				return false
			case *ast.SelectorExpr:
				return false
			}
			identifier, ok := node.(*ast.Ident)
			if !ok || required[identifier.Name] || types[identifier.Name] == nil {
				return true
			}
			required[identifier.Name] = true
			visit(types[identifier.Name].Type)
			return true
		})
	}
	for _, contract := range interfaces {
		visit(contract.Type)
	}
	return required
}

func containsStruct(node ast.Node) bool {
	found := false
	ast.Inspect(node, func(node ast.Node) bool {
		if _, ok := node.(*ast.StructType); ok {
			found = true
		}
		return !found
	})
	return found
}
