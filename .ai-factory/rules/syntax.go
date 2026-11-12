package main

import (
	"go/ast"
	"go/token"
	"strings"
)

func unwrap(expression ast.Expr) ast.Expr {
	for {
		switch value := expression.(type) {
		case *ast.ParenExpr:
			expression = value.X
		case *ast.StarExpr:
			expression = value.X
		case *ast.UnaryExpr:
			if value.Op != token.AND {
				return expression
			}
			expression = value.X
		default:
			return expression
		}
	}
}

func receiver(function *ast.FuncDecl) (string, string) {
	if function.Recv == nil || len(function.Recv.List) != 1 {
		return "", ""
	}
	typ, ok := unwrap(function.Recv.List[0].Type).(*ast.Ident)
	if !ok {
		return "", ""
	}
	name := ""
	if len(function.Recv.List[0].Names) == 1 {
		name = function.Recv.List[0].Names[0].Name
	}
	return typ.Name, name
}

func (repo *repository) typeName(file *sourceFile, expression ast.Expr) string {
	return repo.resolveType(file, expression, 0)
}

func (repo *repository) resolveType(file *sourceFile, expression ast.Expr, depth int) string {
	if depth > 32 {
		return ""
	}
	switch value := expression.(type) {
	case *ast.Ident:
		if definition, ok := repo.types[file.dir][value.Name]; ok {
			if definition.spec.Assign.IsValid() {
				return repo.resolveType(definition.source, definition.spec.Type, depth+1)
			}
			return repo.packagePath(file.dir) + "." + value.Name
		}
		if imported, ok := file.imports["."]; ok {
			return imported + "." + value.Name
		}
		return value.Name
	case *ast.SelectorExpr:
		owner, ok := value.X.(*ast.Ident)
		if ok && file.imports[owner.Name] != "" {
			return file.imports[owner.Name] + "." + value.Sel.Name
		}
	case *ast.ParenExpr:
		return repo.resolveType(file, value.X, depth+1)
	}
	return ""
}

func (repo *repository) runnerFields(file *sourceFile, fields *ast.FieldList, target string) map[string]bool {
	result := map[string]bool{}
	if fields == nil {
		return result
	}
	for _, field := range fields.List {
		if repo.typeName(file, field.Type) != target {
			continue
		}
		for _, name := range field.Names {
			result[name.Name] = true
		}
	}
	return result
}

func agentName(name string) bool {
	base := strings.TrimSuffix(name, "Agent")
	return base != name && base != "" && base[0] >= 'A' && base[0] <= 'Z'
}

func hasCall(node ast.Node, name string) bool {
	if node == nil {
		return false
	}
	found := false
	ast.Inspect(node, func(node ast.Node) bool {
		if found {
			return false
		}
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return !found
		}
		switch value := call.Fun.(type) {
		case *ast.Ident:
			found = value.Name == name
		case *ast.SelectorExpr:
			found = value.Sel.Name == name
		}
		return !found
	})
	return found
}

func expressionType(expression ast.Expr) string {
	value, ok := unwrap(expression).(*ast.Ident)
	if !ok {
		return ""
	}
	return value.Name
}

func (repo *repository) hasAssertion(dir, implementation, target string) bool {
	for _, file := range repo.files {
		if file.test || file.dir != dir {
			continue
		}
		for _, declaration := range file.tree.Decls {
			gen, ok := declaration.(*ast.GenDecl)
			if !ok || gen.Tok != token.VAR {
				continue
			}
			for _, specification := range gen.Specs {
				spec := specification.(*ast.ValueSpec)
				if repo.typeName(file, spec.Type) != target {
					continue
				}
				for index, name := range spec.Names {
					if name.Name != "_" || index >= len(spec.Values) {
						continue
					}
					if assertedType(spec.Values[index]) == implementation {
						return true
					}
				}
			}
		}
	}
	return false
}

func assertedType(expression ast.Expr) string {
	switch value := expression.(type) {
	case *ast.CallExpr:
		if len(value.Args) == 1 {
			return expressionType(value.Fun)
		}
	case *ast.UnaryExpr:
		if literal, ok := value.X.(*ast.CompositeLit); ok {
			return expressionType(literal.Type)
		}
	case *ast.CompositeLit:
		return expressionType(value.Type)
	}
	return ""
}
