# Keep implementation and contract declarations focused

## Scope

This rule applies to non-test, non-generated Go source. It checks top-level
structs throughout `internal/agents/llm`, including the Codex, Claude, and
shared `runnercli` packages. It checks interfaces only in the exact
`internal/agents` package, so provider subpackages can define their own helper
types without becoming agent contract files.

## Requirement

Keep at most one top-level struct in each provider implementation file and at
most one top-level interface in each agent contract file. Name each declaration
file in snake case. Files containing only helper functions are allowed.

## Valid example

```go
// internal/agents/agent_runner.go
type AgentRunner interface { Run() }

// internal/agents/llm/claude/claude_cli_runner.go
type ClaudeCLIRunner struct{}
```

## Run the check

From the project root, run:

```sh
./.ai-factory/rules/checkrules adapter-port-one-struct
```

Configure implementation and contract roots in the settings block at the top
of the Go source.
