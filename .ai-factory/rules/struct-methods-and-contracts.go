package main

import (
	"errors"
	"fmt"
	"go/ast"
	"go/token"
)

// Struct source roots and exclusions are editable for another project layout.
var structFileSettings = struct {
	SourceRoots   []string
	ExcludedRoots []string
}{
	SourceRoots:   []string{"."},
	ExcludedRoots: []string{},
}

func init() { register("struct-methods-and-contracts", checkStructMethodsAndContracts) }

type structDeclaration struct {
	file *sourceFile
	spec *ast.TypeSpec
}

type structDeclarationKey struct {
	directory, packageName, name string
}

type structDeclarations struct {
	local    map[structDeclarationKey]structDeclaration
	imported map[string]structDeclaration
}

func checkStructMethodsAndContracts(repo *repository) error {
	files, err := repo.sources(true)
	if err != nil {
		return err
	}
	declarations := indexStructDeclarations(repo, files)
	var issues []error
	for _, file := range files {
		if !structFileInScope(file.dir) {
			continue
		}
		issues = append(issues, checkStructMethods(file, declarations)...)
		issues = append(issues, checkStructAssertions(repo, file, declarations)...)
	}
	return errors.Join(issues...)
}

func structFileInScope(directory string) bool {
	return matchesAnyRoot(directory, structFileSettings.SourceRoots) &&
		!matchesAnyRoot(directory, structFileSettings.ExcludedRoots)
}

func indexStructDeclarations(repo *repository, files []*sourceFile) structDeclarations {
	declarations := structDeclarations{
		local:    map[structDeclarationKey]structDeclaration{},
		imported: map[string]structDeclaration{},
	}
	for _, file := range files {
		for _, declaration := range file.tree.Decls {
			gen, ok := declaration.(*ast.GenDecl)
			if !ok || gen.Tok != token.TYPE {
				continue
			}
			for _, specification := range gen.Specs {
				spec := specification.(*ast.TypeSpec)
				if _, ok := spec.Type.(*ast.StructType); !ok {
					continue
				}
				value := structDeclaration{file: file, spec: spec}
				key := structDeclarationKey{file.dir, file.tree.Name.Name, spec.Name.Name}
				declarations.local[key] = value
				if !file.test {
					declarations.imported[repo.packagePath(file.dir)+"."+spec.Name.Name] = value
				}
			}
		}
	}
	return declarations
}

func checkStructMethods(file *sourceFile, declarations structDeclarations) []error {
	var issues []error
	for _, declaration := range file.tree.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Recv == nil {
			continue
		}
		name := methodReceiverType(function)
		if name == "" {
			continue
		}
		structure, exists := structDeclaredInFile(file, name)
		if !exists {
			key := structDeclarationKey{file.dir, file.tree.Name.Name, name}
			structure, exists = declarations.local[key]
		}
		if !exists || !structFileInScope(structure.file.dir) {
			continue
		}
		if structure.file != file {
			issues = append(issues, file.at(function.Pos(), fmt.Sprintf("method %s.%s must be declared in %s after the %s struct", name, function.Name.Name, structure.file.name, name)))
			continue
		}
		if function.Pos() <= structure.spec.End() {
			issues = append(issues, file.at(function.Pos(), fmt.Sprintf("method %s.%s must follow the %s struct declaration", name, function.Name.Name, name)))
		}
	}
	return issues
}

func methodReceiverType(function *ast.FuncDecl) string {
	if function.Recv == nil || len(function.Recv.List) != 1 {
		return ""
	}
	identifier, ok := unparameterizedType(function.Recv.List[0].Type).(*ast.Ident)
	if !ok {
		return ""
	}
	return identifier.Name
}

func structDeclaredInFile(file *sourceFile, name string) (structDeclaration, bool) {
	for _, declaration := range file.tree.Decls {
		gen, ok := declaration.(*ast.GenDecl)
		if !ok || gen.Tok != token.TYPE {
			continue
		}
		for _, specification := range gen.Specs {
			spec := specification.(*ast.TypeSpec)
			if spec.Name.Name != name {
				continue
			}
			if _, ok := spec.Type.(*ast.StructType); ok {
				return structDeclaration{file: file, spec: spec}, true
			}
		}
	}
	return structDeclaration{}, false
}

func checkStructAssertions(repo *repository, file *sourceFile, declarations structDeclarations) []error {
	var issues []error
	for _, declaration := range file.tree.Decls {
		gen, ok := declaration.(*ast.GenDecl)
		if !ok || gen.Tok != token.VAR {
			continue
		}
		for _, specification := range gen.Specs {
			spec := specification.(*ast.ValueSpec)
			if spec.Type == nil {
				continue
			}
			for index, name := range spec.Names {
				if name.Name != "_" || index >= len(spec.Values) {
					continue
				}
				structure, exists := assertedStruct(repo, file, spec.Values[index], declarations)
				if !exists || !structFileInScope(structure.file.dir) {
					continue
				}
				if structure.file != file {
					issues = append(issues, file.at(name.Pos(), fmt.Sprintf("conformance assertion for %s must be declared in %s after the %s struct", structure.spec.Name.Name, structure.file.name, structure.spec.Name.Name)))
					continue
				}
				if name.Pos() <= structure.spec.End() {
					issues = append(issues, file.at(name.Pos(), fmt.Sprintf("conformance assertion for %s must follow the %s struct declaration", structure.spec.Name.Name, structure.spec.Name.Name)))
				}
			}
		}
	}
	return issues
}

func assertedStruct(repo *repository, file *sourceFile, expression ast.Expr, declarations structDeclarations) (structDeclaration, bool) {
	typ := assertedStructType(expression)
	if typ == nil {
		return structDeclaration{}, false
	}
	if identifier, ok := typ.(*ast.Ident); ok {
		if structure, exists := structDeclaredInFile(file, identifier.Name); exists {
			return structure, true
		}
		key := structDeclarationKey{file.dir, file.tree.Name.Name, identifier.Name}
		if structure, exists := declarations.local[key]; exists {
			return structure, true
		}
	}
	structure, exists := declarations.imported[repo.typeName(file, typ)]
	return structure, exists
}

func assertedStructType(expression ast.Expr) ast.Expr {
	switch value := expression.(type) {
	case *ast.CallExpr:
		if len(value.Args) == 1 {
			return unparameterizedType(value.Fun)
		}
	case *ast.UnaryExpr:
		if value.Op == token.AND {
			if literal, ok := value.X.(*ast.CompositeLit); ok {
				return unparameterizedType(literal.Type)
			}
		}
	case *ast.CompositeLit:
		return unparameterizedType(value.Type)
	}
	return nil
}

func unparameterizedType(expression ast.Expr) ast.Expr {
	for {
		switch value := expression.(type) {
		case *ast.ParenExpr:
			expression = value.X
		case *ast.StarExpr:
			expression = value.X
		case *ast.IndexExpr:
			expression = value.X
		case *ast.IndexListExpr:
			expression = value.X
		default:
			return expression
		}
	}
}
