# Construct runner-backed agents through injection

## Scope

This rule applies to non-test, non-generated `*.go` files beneath the
configured agent root. It excludes vendored source, hidden metadata paths, and
directories listed in the checker settings.

## Requirement

Every struct that stores the configured runner interface must be named
`{Name}Agent`. Runner calls must occur only in methods of those structs.
`New{Name}Agent` must accept the runner, may accept additional dependencies,
may return additional values (including `error`), return the matching agent,
and store the supplied runner on it. Helpers that do not invoke the runner may
be ordinary functions.

## Valid example

```go
type RuleAgent struct { runner AgentRunner }

func NewRuleAgent(runner AgentRunner, label string) (*RuleAgent, error) {
	return &RuleAgent{runner: runner}, nil
}

var _ Agent = (*RuleAgent)(nil)

func (agent *RuleAgent) Converse(prompt string, onMessage func(string)) (string, error) {
	return agent.runner.Run(".", prompt, onMessage)
}
```

## Run the check

From the project root, run:

```sh
./.ai-factory/rules/checkrules agent-constructor-injection
```

Configure the agent root, runner-interface import and name, and exclusions in
the settings block at the top of the Go source.
