# Keep interface declaration files focused

## Scope

This rule applies to every Go source file, including tests, that declares a
named interface. Generated files, vendored source, hidden metadata paths, and
the configured generated-artifact directory are excluded.

## Requirement

An interface-declaring file may contain imports, named interface declarations,
and non-struct named supporting types used by those interfaces, directly or
transitively. It must not declare structs, functions or methods, variables,
constants, or unrelated helper types. Files that merely use interfaces are not
affected.

## Valid example

```go
package example

type Handler interface {
	Handle(Callback)
}

type Callback func(string) error
```

## Run the check

From the project root, run:

```sh
./.ai-factory/rules/checkrules interface-files
```
