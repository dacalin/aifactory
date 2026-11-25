# Working on AI Factory

- Read [the implemented specification](docs/spec.md) before changing product behavior. Update the affected When/Then scenarios in the same change.
- Read [the architecture](docs/architecture.md) before changing package boundaries or wiring. Use the [dependency manifest](docs/dependencies.yaml) to record new host tools.
- Treat cancellation as a design principle. Pass the caller's context through agent turns and provider runners to the CLI process. Exercise lifecycle changes with a simulated CLI.
- Check [the project rules](.ai-factory/rules/INDEX.md) before editing Go code. Change rule definitions, checker sources, or the rule index only when explicitly asked to change the rules.
- Use `go test ./...` for the application and `go -C .ai-factory/rules test ./...` for the checker module. Use `go run . rule review` to validate the full review command from source. Report sandbox integration skips or setup failures separately from the checker results.
- Keep product specs limited to implemented behavior. Record planned work separately and link to it when it explains a current boundary.
