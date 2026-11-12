package main

import (
	"errors"
	"fmt"
	"go/ast"
	"go/token"
)

// Add each command constructor and its own file here. Helpers may have other files.
var commandFiles = []struct {
	File, Function string
	Constructor    bool
}{
	{"root.go", "NewRootCommand", true},
	{"init.go", "newInitCommand", true},
	{"rule.go", "newRuleCommand", true},
	{"review.go", "newReviewCommand", true},
	{"conversation.go", "runConversation", false},
}
var commandRoot = "internal/commands"
var rootSubcommands = []string{"newInitCommand", "newRuleCommand"}
var cobraCommandType = "github.com/spf13/cobra.Command"

func init() { register("cli-command-separation", checkCLI) }

func namedFunction(file *sourceFile, name string) *ast.FuncDecl {
	for _, declaration := range file.tree.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if ok && function.Recv == nil && function.Name.Name == name {
			return function
		}
	}
	return nil
}

func checkCLI(repo *repository) error {
	files, err := repo.sources(false)
	if err != nil {
		return err
	}
	var issues []error
	byName := map[string]*sourceFile{}
	for _, file := range files {
		byName[file.name] = file
		if file.name == "main.go" {
			issues = append(issues, checkMain(file))
		}
		if within(file.dir, commandRoot) || file.name == bootstrapSettings.File || file.name == "main.go" {
			for _, imported := range file.imports {
				if imported == "flag" || imported == "github.com/spf13/pflag" {
					issues = append(issues, fmt.Errorf("%s: use Cobra for CLI parsing", file.name))
				}
			}
		}
	}
	if byName["main.go"] == nil {
		issues = append(issues, fmt.Errorf("missing main.go"))
	}
	issues = append(issues, checkCommandFiles(repo, files, byName))
	root := byName[commandRoot+"/root.go"]
	if root != nil {
		for _, constructor := range rootSubcommands {
			if !addsCommand(root.tree, constructor) {
				issues = append(issues, fmt.Errorf("%s: root must register %s with AddCommand", root.name, constructor))
			}
		}
	}
	rule := byName[commandRoot+"/rule.go"]
	if rule != nil && !addsCommand(rule.tree, "newReviewCommand") {
		issues = append(issues, fmt.Errorf("%s: rule must register newReviewCommand with AddCommand", rule.name))
	}
	bootstrap := byName[bootstrapSettings.File]
	if bootstrap != nil && !hasCall(bootstrap.tree, "NewRootCommand") {
		issues = append(issues, fmt.Errorf("%s must construct the root command", bootstrap.name))
	}
	return errors.Join(issues...)
}

func checkMain(file *sourceFile) error {
	var issues []error
	main := namedFunction(file, "main")
	if main == nil || !hasCall(main.Body, "bootstrap") || !hasCall(main.Body, "Execute") {
		return fmt.Errorf("%s must execute the bootstrapped root command", file.name)
	}
	for _, declaration := range file.tree.Decls {
		if declaration == main {
			continue
		}
		gen, ok := declaration.(*ast.GenDecl)
		if !ok || gen.Tok != token.IMPORT {
			issues = append(issues, fmt.Errorf("%s must contain only main and imports", file.name))
		}
	}
	ast.Inspect(main.Body, func(node ast.Node) bool {
		switch value := node.(type) {
		case *ast.ForStmt, *ast.RangeStmt:
			issues = append(issues, fmt.Errorf("%s: move conversation loops into command handlers", file.name))
		case *ast.SelectorExpr:
			if owner, ok := value.X.(*ast.Ident); ok && file.imports[owner.Name] == "os" && value.Sel.Name == "Args" {
				issues = append(issues, file.at(value.Pos(), "Cobra must parse command arguments"))
			}
		}
		return true
	})
	return errors.Join(issues...)
}

func checkCommandFiles(repo *repository, files []*sourceFile, byName map[string]*sourceFile) error {
	var issues []error
	expected := map[string]string{}
	for _, command := range commandFiles {
		name := commandRoot + "/" + command.File
		expected[command.Function] = name
		file := byName[name]
		if file == nil {
			issues = append(issues, fmt.Errorf("missing command file %s", name))
			continue
		}
		function := namedFunction(file, command.Function)
		if function == nil {
			issues = append(issues, fmt.Errorf("%s must declare %s", name, command.Function))
			continue
		}
		if command.Constructor && !returnsCommand(repo, file, function) {
			issues = append(issues, fmt.Errorf("%s: %s must return *cobra.Command", name, command.Function))
		}
	}
	for _, file := range files {
		if !within(file.dir, commandRoot) {
			continue
		}
		for _, declaration := range file.tree.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if ok && returnsCommand(repo, file, function) && expected[function.Name.Name] != file.name {
				issues = append(issues, fmt.Errorf("%s: give command %s its own configured file", file.name, function.Name.Name))
			}
		}
	}
	return errors.Join(issues...)
}

func returnsCommand(repo *repository, file *sourceFile, function *ast.FuncDecl) bool {
	if function.Type.Results == nil || len(function.Type.Results.List) != 1 {
		return false
	}
	pointer, ok := function.Type.Results.List[0].Type.(*ast.StarExpr)
	return ok && repo.typeName(file, pointer.X) == cobraCommandType
}

func addsCommand(node ast.Node, constructor string) bool {
	found := false
	ast.Inspect(node, func(node ast.Node) bool {
		if found {
			return false
		}
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		method, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || method.Sel.Name != "AddCommand" {
			return true
		}
		for _, argument := range call.Args {
			nested, ok := argument.(*ast.CallExpr)
			if !ok {
				continue
			}
			name, ok := nested.Fun.(*ast.Ident)
			if ok && name.Name == constructor {
				found = true
			}
		}
		return !found
	})
	return found
}
