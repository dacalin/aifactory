# Keep dependency injection in `bootstrap.go`

## Scope

This rule applies to non-test, non-generated Go source in the configured source
paths. It excludes vendored source and hidden metadata paths. The root command
files `main.go` and `bootstrap.go` are always checked.

## Requirement

The composition root is the root-level `bootstrap.go` file, and it must declare
`package main`. It selects and injects the concrete Codex and Claude providers,
and it is the only included file that may directly import their configured
package roots. `main.go` must not import a concrete provider.

The checker treats such imports as the measurable dependency-injection wiring
boundary. Provider options in `internal/agents/llm` and shared helpers in
`internal/agents/llm/runnercli` are not concrete provider roots, so provider
packages may import them. Add every concrete provider root to the settings when
introducing another provider.

## Valid example

```go
// bootstrap.go
package main

import (
	"example.com/project/internal/agents/llm"
	"example.com/project/internal/agents/llm/claude"
	"example.com/project/internal/agents/llm/codex"
)

func bootstrap(provider, prompt string) error {
	options := []llm.Option{llm.WithWritableDirs(".ai-factory/rules")}
	switch provider {
	case "claude":
		return runConversation(claude.NewRunner("", "", options...))
	default:
		return runConversation(codex.NewRunner("", "", options...))
	}
}
```

```go
// main.go
package main

func main() {
	_ = bootstrap("codex", "create a rule")
}
```

## Run the check

From the project root, run:

```sh
./.ai-factory/rules/checkrules dependency-injection-bootstrap
```

Configure the source paths, exclusions, and concrete provider package roots in
the settings block at the top of the Go source.
