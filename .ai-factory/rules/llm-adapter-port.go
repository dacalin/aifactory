package main

import (
	"errors"
	"fmt"
	"go/ast"
	"path/filepath"
	"strconv"
	"strings"
)

// Model integration locations and recognized provider entry points.
var modelRoot = "internal/llm"
var modelCLIs = map[string]bool{"codex": true, "claude": true, "gemini": true, "ollama": true, "aider": true, "opencode": true, "cursor-agent": true}
var modelSDKPrefixes = []string{
	"github.com/openai/", "github.com/anthropics/", "github.com/sashabaranov/go-openai",
	"google.golang.org/genai", "github.com/google/generative-ai-go", "github.com/ollama/ollama",
	"github.com/cohere-ai/", "github.com/mistralai/", "github.com/groq/", "github.com/togethercomputer/",
	"github.com/deepseek-ai/", "github.com/xai-org/",
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime",
	"github.com/Azure/azure-sdk-for-go/sdk/ai/azopenai", "cloud.google.com/go/vertexai",
}
var modelEndpointFragments = []string{
	"api.openai.com", "api.anthropic.com", "generativelanguage.googleapis.com",
	"aiplatform.googleapis.com", "api.mistral.ai", "api.cohere.ai", "api.groq.com",
	"api.together.xyz", "api.perplexity.ai", "api.deepseek.com", "api.x.ai",
	"localhost:11434", "127.0.0.1:11434", ".openai.azure.com",
}

func init() { register("llm-adapter-port", checkModelAccess) }

func checkModelAccess(repo *repository) error {
	files, err := repo.sources(false)
	if err != nil {
		return err
	}
	var issues []error
	found := false
	for _, file := range files {
		if !modelIntegration(file) {
			continue
		}
		found = true
		if !within(file.dir, modelRoot) {
			issues = append(issues, fmt.Errorf("%s: model access belongs under %s", file.name, modelRoot))
			continue
		}
		if !runnerAssertion(repo, file.dir) {
			issues = append(issues, fmt.Errorf("%s: model implementations need an agents.AgentRunner conformance assertion", file.name))
		}
	}
	if err := errors.Join(issues...); err != nil {
		return err
	}
	if found {
		return repo.compile()
	}
	return nil
}

func runnerAssertion(repo *repository, dir string) bool {
	target := repo.packagePath(agentSettings.Root) + "." + agentSettings.Runner
	for name, definition := range repo.types[dir] {
		if _, ok := definition.spec.Type.(*ast.StructType); ok && repo.hasAssertion(dir, name, target) {
			return true
		}
	}
	return false
}

func modelIntegration(file *sourceFile) bool {
	for _, imported := range file.imports {
		for _, prefix := range modelSDKPrefixes {
			if strings.HasPrefix(imported, prefix) {
				return true
			}
		}
	}
	found := false
	ast.Inspect(file.tree, func(node ast.Node) bool {
		if found {
			return false
		}
		switch value := node.(type) {
		case *ast.BasicLit:
			text, err := strconv.Unquote(value.Value)
			if err == nil && (strings.Contains(text, "https://") || strings.Contains(text, "http://")) {
				for _, endpoint := range modelEndpointFragments {
					if strings.Contains(text, endpoint) {
						found = true
					}
				}
			}
		case *ast.CallExpr:
			found = modelCLI(file, value)
		}
		return !found
	})
	return found
}

func modelCLI(file *sourceFile, call *ast.CallExpr) bool {
	function, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	owner, ok := function.X.(*ast.Ident)
	if !ok || file.imports[owner.Name] != "os/exec" {
		return false
	}
	argument := 0
	switch function.Sel.Name {
	case "Command":
	case "CommandContext":
		argument = 1
	default:
		return false
	}
	if len(call.Args) <= argument {
		return false
	}
	literal, ok := call.Args[argument].(*ast.BasicLit)
	if !ok {
		return false
	}
	name, err := strconv.Unquote(literal.Value)
	return err == nil && modelCLIs[filepath.Base(name)]
}
