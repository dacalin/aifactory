# Separate CLI command responsibilities

## Scope

This rule applies to Go source in the configured CLI entrypoint,
bootstrap, and command-handler paths. Test files, generated files, vendored
source, and hidden metadata paths are excluded.

## Requirement

Use Cobra to declare and parse Go CLI commands and flags. The entrypoint must
only execute the bootstrapped root command and report an execution error. It
must not access `os.Args`, use flag-parsing packages, declare command handlers,
or run an interactive conversation loop.

Keep dependency wiring in the configured bootstrap function. Keep each
configured command constructor or handler in its configured file below the
command-handler root. The configured root assembler adds configured subcommands;
shared interactive streaming and follow-up input belong in its configured
conversation handler. These checks are structural guardrails, not proof of all
CLI design or clean-code properties.

## Valid example

```go
// internal/commands/init.go
func newInitCommand() *cobra.Command {
	return &cobra.Command{Use: "init", RunE: runInit}
}
```

```go
// main.go
func main() {
	if err := bootstrap().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}
```

## Run the check

From the project root, run:

```sh
./.ai-factory/rules/checkrules cli-command-separation
```

Configure project paths, expected command-to-file mappings, constructors, and
the bootstrap wiring call in the settings block at the top of the Go source.
