# Go readability limits

## Scope

This rule applies to Go files in the configured source
paths. It excludes test files (`*_test.go`), generated files, vendored source,
and hidden metadata paths.

Keep each included file at most 300 physical lines. Keep each named function
or method at most 60 physical lines, including its signature. Keep its
GoCognit cognitive complexity at most 15. A value equal to a limit passes.

## Valid example

```go
func greeting(name string) string {
	if name == "" {
		return "Hello"
	}
	return "Hello, " + name
}
```

## Run the check

From the project root, run:

```sh
./.ai-factory/rules/checkrules go-readability-limits
```

The shared binary uses the pinned GoCognit library for complexity. Paths and
thresholds are editable in go-readability-limits.go; dependencies are pinned
in the checker's go.mod.
