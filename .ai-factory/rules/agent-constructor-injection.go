package main

import (
	"errors"
	"fmt"
	"go/ast"
)

// The runner contract is owned by the consumer package.
var agentSettings = struct{ Root, Runner string }{"internal/agents", "AgentRunner"}

func init() { register("agent-constructor-injection", checkAgentConstruction) }

func checkAgentConstruction(repo *repository) error {
	files, err := repo.sources(false)
	if err != nil {
		return err
	}
	target := repo.packagePath(agentSettings.Root) + "." + agentSettings.Runner
	var issues []error
	for _, file := range files {
		if !within(file.dir, agentSettings.Root) {
			continue
		}
		for _, declaration := range file.tree.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if ok {
				issues = append(issues, checkRunnerCalls(repo, file, function, target))
			}
		}
	}
	for dir, definitions := range repo.types {
		if !within(dir, agentSettings.Root) {
			continue
		}
		for name, definition := range definitions {
			structure, ok := definition.spec.Type.(*ast.StructType)
			if !ok {
				continue
			}
			fields := repo.runnerFields(definition.source, structure.Fields, target)
			if len(fields) == 0 {
				continue
			}
			if !agentName(name) {
				issues = append(issues, definition.source.at(definition.spec.Pos(), fmt.Sprintf("runner-backed struct %s must use {Name}Agent naming", name)))
				continue
			}
			constructorFile, constructor := findFunction(files, dir, "New"+name)
			if constructor == nil {
				issues = append(issues, fmt.Errorf("%s: %s needs New%s with an injected runner", definition.source.name, name, name))
				continue
			}
			parameters := repo.runnerFields(constructorFile, constructor.Type.Params, target)
			if len(parameters) == 0 || !constructorReturns(repo, constructorFile, constructor, name) || !storesRunner(constructor, name, structure, fields, parameters) {
				issues = append(issues, constructorFile.at(constructor.Pos(), fmt.Sprintf("New%s must receive and store %s on its returned agent", name, agentSettings.Runner)))
			}
		}
	}
	return errors.Join(issues...)
}

func findFunction(files []*sourceFile, dir, name string) (*sourceFile, *ast.FuncDecl) {
	for _, file := range files {
		if file.dir != dir {
			continue
		}
		if function := namedFunction(file, name); function != nil {
			return file, function
		}
	}
	return nil, nil
}

func constructorReturns(repo *repository, file *sourceFile, constructor *ast.FuncDecl, name string) bool {
	if constructor.Type.Results == nil {
		return false
	}
	for _, result := range constructor.Type.Results.List {
		typ := repo.typeName(file, unwrap(result.Type))
		if typ == repo.packagePath(file.dir)+"."+name || typ == repo.packagePath(agentSettings.Root)+".Agent" {
			return true
		}
	}
	return false
}

func checkRunnerCalls(repo *repository, file *sourceFile, function *ast.FuncDecl, target string) error {
	if function.Body == nil {
		return nil
	}
	class, receiverVariable := receiver(function)
	fields := map[string]bool{}
	if definition, exists := repo.types[file.dir][class]; exists {
		if structure, ok := definition.spec.Type.(*ast.StructType); ok {
			fields = repo.runnerFields(definition.source, structure.Fields, target)
		}
	}
	bindings := repo.runnerFields(file, function.Type.Params, target)
	var issues []error
	ast.Inspect(function.Body, func(node ast.Node) bool {
		if assignment, ok := node.(*ast.AssignStmt); ok {
			for index, value := range assignment.Rhs {
				if index >= len(assignment.Lhs) || !runnerExpression(value, bindings, receiverVariable, fields) {
					continue
				}
				if name, ok := assignment.Lhs[index].(*ast.Ident); ok {
					bindings[name.Name] = true
				}
			}
		}
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		method, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || method.Sel.Name != "Run" || !runnerExpression(method.X, bindings, receiverVariable, fields) {
			return true
		}
		if !agentName(class) || len(fields) == 0 {
			issues = append(issues, file.at(call.Pos(), "runner calls belong to methods of constructor-injected {Name}Agent structs"))
		}
		return true
	})
	return errors.Join(issues...)
}

func runnerExpression(expression ast.Expr, bindings map[string]bool, receiver string, fields map[string]bool) bool {
	switch value := expression.(type) {
	case *ast.Ident:
		return bindings[value.Name]
	case *ast.SelectorExpr:
		owner, ok := value.X.(*ast.Ident)
		return ok && owner.Name == receiver && fields[value.Sel.Name]
	case *ast.ParenExpr:
		return runnerExpression(value.X, bindings, receiver, fields)
	}
	return false
}

func storesRunner(constructor *ast.FuncDecl, name string, structure *ast.StructType, fields, parameters map[string]bool) bool {
	if constructor.Body == nil {
		return false
	}
	instances := map[string]bool{}
	returned, valid := false, true
	ast.Inspect(constructor.Body, func(node ast.Node) bool {
		switch value := node.(type) {
		case *ast.AssignStmt:
			updateInstances(value, name, structure, fields, parameters, instances)
		case *ast.ReturnStmt:
			if len(value.Results) == 0 {
				return true
			}
			expression := unwrap(value.Results[0])
			if ident, ok := expression.(*ast.Ident); ok && ident.Name == "nil" {
				return true
			}
			if literal, ok := expression.(*ast.CompositeLit); ok && expressionType(literal.Type) == name {
				returned = true
				valid = valid && literalStores(literal, structure, fields, parameters)
			} else if ident, ok := expression.(*ast.Ident); ok {
				injected, exists := instances[ident.Name]
				if exists {
					returned = true
					valid = valid && injected
				}
			}
		}
		return true
	})
	return returned && valid
}

func updateInstances(assignment *ast.AssignStmt, name string, structure *ast.StructType, fields, parameters, instances map[string]bool) {
	for index, expression := range assignment.Rhs {
		if index >= len(assignment.Lhs) {
			continue
		}
		left := assignment.Lhs[index]
		if literal, ok := unwrap(expression).(*ast.CompositeLit); ok && expressionType(literal.Type) == name {
			if variable, ok := left.(*ast.Ident); ok {
				instances[variable.Name] = literalStores(literal, structure, fields, parameters)
			}
		}
		selected, ok := left.(*ast.SelectorExpr)
		if !ok || !fields[selected.Sel.Name] {
			continue
		}
		owner, ownerOK := selected.X.(*ast.Ident)
		provided, providedOK := expression.(*ast.Ident)
		if ownerOK && providedOK && parameters[provided.Name] {
			if _, exists := instances[owner.Name]; exists {
				instances[owner.Name] = true
			}
		}
	}
}

func literalStores(literal *ast.CompositeLit, structure *ast.StructType, fields, parameters map[string]bool) bool {
	var ordered []string
	for _, field := range structure.Fields.List {
		for _, name := range field.Names {
			ordered = append(ordered, name.Name)
		}
	}
	for index, value := range literal.Elts {
		fieldName := ""
		if pair, ok := value.(*ast.KeyValueExpr); ok {
			if key, ok := pair.Key.(*ast.Ident); ok {
				fieldName = key.Name
			}
			value = pair.Value
		} else if index < len(ordered) {
			fieldName = ordered[index]
		}
		provided, ok := value.(*ast.Ident)
		if ok && fields[fieldName] && parameters[provided.Name] {
			return true
		}
	}
	return false
}
