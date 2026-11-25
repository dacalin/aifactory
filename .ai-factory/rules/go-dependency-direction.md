# Depend on consumer-owned interfaces

## Scope

The checker resolves direct imports for the active Go build. It treats the exact
`internal/agents` package as the agent-contract boundary and treats
`internal/agents/llm` and all of its descendants as the provider boundary.

## Requirement

`internal/agents` owns the agent contracts and must not import
`internal/agents/llm` or any of its subpackages. Codex and Claude providers may
import `agents` and `runnercli`, but neither they nor other packages under
`internal/agents/llm` may import `internal/commands`. Bootstrap connects
concrete providers to consumers.

## Valid example

```go
// internal/agents/llm/codex/codex_cli_runner.go
package codex

import (
	"context"

	"example.com/project/internal/agents"
	"example.com/project/internal/agents/llm/runnercli"
)

type CodexCLIRunner struct{ cli *runnercli.CLI }

var _ agents.AgentRunner = (*CodexCLIRunner)(nil)

func (*CodexCLIRunner) Run(context.Context, string, string, func(string)) (string, error) {
	return "", nil
}
```

## Run the check

From the project root, run:

```sh
./.ai-factory/rules/checkrules go-dependency-direction
```

Configure forbidden source and target roots, including whether a source root
includes descendants, in the settings block at the top of the Go source.
