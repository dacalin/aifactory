package main

import (
	"errors"
	"fmt"
	"go/ast"
)

func init() { register("agent-converse-interface", checkAgentInterface) }

func checkAgentInterface(repo *repository) error {
	files, err := repo.sources(false)
	if err != nil {
		return err
	}
	definition, exists := repo.types[agentSettings.Root]["Agent"]
	if !exists {
		return fmt.Errorf("%s must define shared interface Agent", agentSettings.Root)
	}
	contract, ok := definition.spec.Type.(*ast.InterfaceType)
	if !ok || !conversationContract(contract) {
		return fmt.Errorf("Agent must expose Converse(string, func(string)) (string, error)")
	}
	var issues []error
	target := repo.packagePath(agentSettings.Root) + ".Agent"
	for _, file := range files {
		if !within(file.dir, agentSettings.Root) {
			continue
		}
		for _, name := range declaredNames(file.tree, "struct") {
			if agentName(name) && !repo.hasAssertion(file.dir, name, target) {
				issues = append(issues, fmt.Errorf("%s: add explicit Agent conformance assertion for %s", file.name, name))
			}
		}
	}
	if err := errors.Join(issues...); err != nil {
		return err
	}
	return repo.compile()
}

func conversationContract(contract *ast.InterfaceType) bool {
	for _, field := range contract.Methods.List {
		if len(field.Names) != 1 || field.Names[0].Name != "Converse" {
			continue
		}
		method, ok := field.Type.(*ast.FuncType)
		if !ok || method.Params == nil || method.Results == nil || len(method.Params.List) != 2 || len(method.Results.List) != 2 {
			return false
		}
		callback, ok := method.Params.List[1].Type.(*ast.FuncType)
		return identifier(method.Params.List[0].Type, "string") &&
			ok && callback.Params != nil && len(callback.Params.List) == 1 &&
			identifier(callback.Params.List[0].Type, "string") &&
			(callback.Results == nil || len(callback.Results.List) == 0) &&
			identifier(method.Results.List[0].Type, "string") &&
			identifier(method.Results.List[1].Type, "error")
	}
	return false
}

func identifier(expression ast.Expr, name string) bool {
	value, ok := expression.(*ast.Ident)
	return ok && value.Name == name
}
