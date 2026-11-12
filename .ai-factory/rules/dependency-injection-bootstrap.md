# Keep dependency injection in `bootstrap.go`

## Scope

This rule applies to non-test, non-generated Go source in the configured source
paths. It excludes vendored source and hidden metadata paths. The root command
files `main.go` and `bootstrap.go` are always checked.

## Requirement

The composition root is the root-level `bootstrap.go` file, and it must declare
`package main`. It is the only included file that may directly import a
project-local concrete implementation package under the configured implementation
roots. `main.go` must not import a concrete implementation.

The checker treats such imports as the measurable dependency-injection wiring
boundary. Add every package root containing concrete implementations to its
settings so new kinds of infrastructure remain covered.

## Valid example

```go
// bootstrap.go
package main

import "example.com/project/internal/llm"

func bootstrap(prompt string) error {
	return runConversation(&llm.Runner{}, prompt)
}
```

```go
// main.go
package main

func main() {
	_ = bootstrap("create a rule")
}
```

## Run the check

From the project root, run:

```sh
./.ai-factory/rules/checkrules dependency-injection-bootstrap
```

Configure the source paths, exclusions, and concrete implementation package
roots in the settings block at the top of the Go source.
