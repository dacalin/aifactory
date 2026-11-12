# Make agents conform to the conversation interface

## Scope

This rule applies to non-test, non-generated `*.go` files beneath the
configured agent root. It excludes vendored source, hidden metadata paths, and
directories listed in the checker settings.

## Requirement

Every `{Name}Agent` struct must implement the configured shared `Agent`
interface. It must declare `Converse(prompt string, onMessage func(string))
(string, error)` and an explicit compile-time assertion, for example
`var _ Agent = (*RuleAgent)(nil)`. When the interface is imported from another
agent package, the assertion may use that package's qualified `Agent` name.

## Valid example

```go
type RuleAgent struct{}

var _ Agent = (*RuleAgent)(nil)

func (agent *RuleAgent) Converse(prompt string, onMessage func(string)) (string, error) {
	return "", nil
}
```

## Run the check

From the project root, run:

```sh
./.ai-factory/rules/checkrules agent-converse-interface
```

Configure the agent root, shared interface import and name, and exclusions in
the settings block at the top of the Go source.
