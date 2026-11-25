# Centralize direct Git execution

## Scope

This rule applies to Go source and tests beneath the configured source roots.
Generated files, vendored source, hidden metadata paths, generated rule
artifacts, and configured exclusions are not scanned. It recognizes direct
standard-library `os/exec` calls to the configured Git executable names through
`Command`, `CommandContext`, and `LookPath`, including import aliases.

## Requirement

The Git execution boundary owns the direct process-level interaction with the
Git executable. Its configured implementation root is
`internal/workspace/git`; every direct recognized Git invocation must be in
that directory or one of its descendants, including test helpers. Other
packages use a contract or higher-level adapter instead of starting or locating
Git themselves.

Keep the Git executable name as a direct string literal in those `os/exec`
calls so the boundary remains verifiable. Configure the execution root, source
roots, exclusions, and recognized executable names in the settings block at
the top of the checker source.

## Valid example

```go
// internal/workspace/git/status.go
package git

import (
	"context"
	"os/exec"
)

func status(ctx context.Context, repository string) error {
	return exec.CommandContext(ctx, "git", "-C", repository, "status", "--porcelain").Run()
}
```

## Run the check

From the project root, run:

```sh
./.ai-factory/rules/checkrules git-execution-workspace
```
