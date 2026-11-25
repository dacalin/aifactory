# Organize agents and provider implementations by responsibility

## Scope

This rule applies to non-test, non-generated Go source in the configured agent,
provider-option, provider, and shared CLI package roots. Retired ports and
adapters roots must contain no Go source.

## Requirement

`internal/agents` owns `RuleAgent`, `Agent`, and `AgentRunner`.
`internal/agents/llm` owns the `Option` type used to configure providers.
The Codex and Claude packages each contain a struct that explicitly asserts
`agents.AgentRunner`; `internal/agents/llm/runnercli` owns their shared CLI
infrastructure.

## Valid example

```go
// internal/agents/llm/claude/claude_cli_runner.go
package claude

import (
	"context"

	"example.com/project/internal/agents"
)

type ClaudeCLIRunner struct{}

var _ agents.AgentRunner = (*ClaudeCLIRunner)(nil)

func (*ClaudeCLIRunner) Run(context.Context, string, string, func(string)) (string, error) {
	return "", nil
}
```

## Run the check

From the project root, run:

```sh
./.ai-factory/rules/checkrules package-organization
```

Configure all package roots and required contract names in the settings block at
the top of the Go source.
