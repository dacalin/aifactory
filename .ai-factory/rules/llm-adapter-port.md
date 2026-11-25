# Provider implementations satisfy consumer-owned contracts

## Scope

This rule applies to non-test, non-generated Go files that directly invoke a
recognized provider CLI, SDK, or endpoint. Provider integrations belong below
`internal/agents/llm`, including `codex` and `claude`.

## Requirement

Keep provider SDK, endpoint, and CLI calls under `internal/agents/llm`.
Every package with a concrete provider runner must assert conformance to the
`agents.AgentRunner` contract it implements.

## Valid example

```go
package codex

import (
	"context"

	"example.com/project/internal/agents"
)

type CodexCLIRunner struct{}

var _ agents.AgentRunner = (*CodexCLIRunner)(nil)

func (*CodexCLIRunner) Run(context.Context, string, string, func(string)) (string, error) {
	return "", nil
}
```

## Run the check

From the project root, run:

```sh
./.ai-factory/rules/checkrules llm-adapter-port
```

Configure the interface-owning package, provider root, and recognized access
patterns in the settings block at the top of the Go source. The checker runs Go
tests to compile assertions when it finds provider access. Extend provider
detection when introducing an unrecognized SDK, CLI, or endpoint proxy.
