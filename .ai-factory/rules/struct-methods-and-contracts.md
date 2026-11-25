# Co-locate struct methods and conformance assertions

## Scope

This rule applies to Go source, including tests, beneath the configured source
roots. Generated files, vendored source, hidden metadata paths, generated rule
artifacts, and configured exclusions are not scanned.

## Requirement

Every receiver method of a named struct must be declared in the file that
declares that struct and after its declaration. An explicit compile-time
conformance assertion for that struct must be in the same location too, for
example `var _ workspace.Manager = (*Manager)(nil)`.

The rule applies to receiver methods and explicit assertions only. It does not
require plain data structs to implement an interface or treat ordinary functions
as constructors.

## Valid example

```go
package git

type Manager struct{}

var _ workspace.Manager = (*Manager)(nil)

func (manager *Manager) Create() error {
	return nil
}
```

## Run the check

From the project root, run:

```sh
./.ai-factory/rules/checkrules struct-methods-and-contracts
```

Configure source roots and exclusions in the settings block at the top of
`struct-methods-and-contracts.go`.
