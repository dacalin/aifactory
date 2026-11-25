package main

import (
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"path"
	"strconv"
	"strings"
)

// The Git execution boundary and scan scope are editable for another project layout.
type gitExecutionConfiguration struct {
	ExecutionRoot   string
	SourceRoots     []string
	ExcludedRoots   []string
	ExecutableNames []string
}

var gitExecutionSettings = gitExecutionConfiguration{
	ExecutionRoot:   "internal/workspace/git",
	SourceRoots:     []string{"."},
	ExcludedRoots:   []string{},
	ExecutableNames: []string{"git", "git.exe"},
}

func init() { register("git-execution-workspace", checkGitExecution) }

func checkGitExecution(repo *repository) error {
	files, err := repo.sources(true)
	if err != nil {
		return err
	}
	var issues []error
	for _, file := range files {
		if !gitExecutionInScope(file.dir) || within(file.dir, gitExecutionSettings.ExecutionRoot) {
			continue
		}
		ast.Inspect(file.tree, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			if program, found := gitCallProgram(file, call); found && namedGitExecutable(program) {
				issues = append(issues, file.at(program.Pos(), fmt.Sprintf("direct Git invocation must be located under %s", gitExecutionSettings.ExecutionRoot)))
			}
			return true
		})
	}
	return errors.Join(issues...)
}

func gitExecutionInScope(directory string) bool {
	return matchesAnyRoot(directory, gitExecutionSettings.SourceRoots) && !matchesAnyRoot(directory, gitExecutionSettings.ExcludedRoots)
}

func gitCallProgram(file *sourceFile, call *ast.CallExpr) (ast.Expr, bool) {
	switch function := call.Fun.(type) {
	case *ast.SelectorExpr:
		owner, ok := function.X.(*ast.Ident)
		if !ok || file.imports[owner.Name] != "os/exec" {
			return nil, false
		}
		return execProgramArgument(function.Sel.Name, call.Args)
	case *ast.Ident:
		if file.imports["."] != "os/exec" {
			return nil, false
		}
		return execProgramArgument(function.Name, call.Args)
	}
	return nil, false
}

func execProgramArgument(function string, arguments []ast.Expr) (ast.Expr, bool) {
	position := 0
	switch function {
	case "Command", "LookPath":
	case "CommandContext":
		position = 1
	default:
		return nil, false
	}
	if len(arguments) <= position {
		return nil, false
	}
	return arguments[position], true
}

func namedGitExecutable(expression ast.Expr) bool {
	literal, ok := unwrap(expression).(*ast.BasicLit)
	if !ok || literal.Kind != token.STRING {
		return false
	}
	executable, err := strconv.Unquote(literal.Value)
	if err != nil {
		return false
	}
	executable = path.Base(strings.ReplaceAll(executable, "\\", "/"))
	for _, name := range gitExecutionSettings.ExecutableNames {
		if strings.EqualFold(executable, name) {
			return true
		}
	}
	return false
}
