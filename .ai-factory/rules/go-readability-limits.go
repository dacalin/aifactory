package main

import (
	"errors"
	"fmt"
	"go/ast"

	"github.com/uudashr/gocognit"
)

// Limits include signatures and physical blank/comment lines.
var readabilityLimits = struct{ FileLines, FunctionLines, CognitiveComplexity int }{300, 60, 15}

func init() { register("go-readability-limits", checkReadability) }

func checkReadability(repo *repository) error {
	files, err := repo.sources(false)
	if err != nil {
		return err
	}
	var issues []error
	for _, file := range files {
		if file.lines > readabilityLimits.FileLines {
			issues = append(issues, fmt.Errorf("%s: %d lines exceeds the %d-file-line limit", file.name, file.lines, readabilityLimits.FileLines))
		}
		for _, declaration := range file.tree.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Body == nil {
				continue
			}
			start := file.tokens.PositionFor(function.Pos(), false).Line
			end := file.tokens.PositionFor(function.End(), false).Line
			if lines := end - start + 1; lines > readabilityLimits.FunctionLines {
				issues = append(issues, file.at(function.Pos(), fmt.Sprintf("%s has %d lines; maximum %d", function.Name.Name, lines, readabilityLimits.FunctionLines)))
			}
			if complexity := gocognit.Complexity(function); complexity > readabilityLimits.CognitiveComplexity {
				issues = append(issues, file.at(function.Pos(), fmt.Sprintf("%s has cognitive complexity %d; maximum %d", function.Name.Name, complexity, readabilityLimits.CognitiveComplexity)))
			}
		}
	}
	return errors.Join(issues...)
}
