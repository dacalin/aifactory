# Name interfaces by role

## Scope

This rule applies to named Go interface declarations, including declarations in
tests, beneath the configured source roots. Generated files, vendored source,
hidden metadata paths, configured exclusions, and rule artifacts are excluded.

## Requirement

An interface name must not contain `Interface`, regardless of case. Name the
contract for the role it represents instead.

## Valid example

```go
type AgentRunner interface {
	Run(projectDir, prompt string, onMessage func(string)) (string, error)
}
```

## Run the check

From the project root, run:

```sh
./.ai-factory/rules/checkrules interface-type-names
```

Configure the scanned source roots, exclusions, and prohibited name fragments
in the settings block at the top of the Go source.
