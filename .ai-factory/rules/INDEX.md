# Project rules

The ready-to-run binary lives alongside the rules. Rebuild it from the project
root after changing checker sources.

```sh
go -C .ai-factory/rules build -o checkrules .
```

The runner reads artifact and rule paths from `factory.yaml`. Managed runs use
`AIFACTORY_PROJECT_DIR`; use `--root` to select a project directly, or
`--data-dir` and `--rules-dir` to override these paths.

Use the CLI from this checkout or worktree (including subdirectories). It runs
the checks.build (when configured) and checks.run commands in factory.yaml.
They use this same entry point; the application does not infer its language.
It returns nonzero if building or checking fails and never starts an LLM.

```sh
aifactory rule review --list
aifactory rule review interface-files
aifactory rule review
```

List checks, run one, or run all directly.

```sh
./.ai-factory/rules/checkrules --list
./.ai-factory/rules/checkrules agent-converse-interface
./.ai-factory/rules/checkrules all
```

- [Keep declarations focused](adapter-port-one-struct.md) — Keep one named provider implementation struct per file below `internal/agents/llm` and one named agent interface per file in `internal/agents`, with matching filenames; allow helper files. [Check](adapter-port-one-struct.go)
  Run: `./.ai-factory/rules/checkrules adapter-port-one-struct`
- [Construct runner-backed agents through injection](agent-constructor-injection.md) — In configured agent Go code, name runner-backed structs `{Name}Agent`; make runner calls only from their methods and inject/store the runner through `New{Name}Agent`. [Check](agent-constructor-injection.go)
  Run: `./.ai-factory/rules/checkrules agent-constructor-injection`
- [Make agents conform to the conversation interface](agent-converse-interface.md) — In configured agent Go code, give every `{Name}Agent` the shared `Converse(prompt string, onMessage func(string)) (string, error)` method and an explicit `Agent` conformance assertion. [Check](agent-converse-interface.go)
  Run: `./.ai-factory/rules/checkrules agent-converse-interface`
- [Separate CLI command responsibilities](cli-command-separation.md) — Use Cobra; keep root execution and error reporting in `main.go`, dependency wiring in `bootstrap.go`, and each configured command or conversation handler in its configured command file. [Check](cli-command-separation.go)
  Run: `./.ai-factory/rules/checkrules cli-command-separation`
- [Keep direct database access inside persistence](database-access-storage.md) — In configured Go source, including tests, keep database APIs under the persistence boundary, let only configured composition files import concrete adapters, and require configured adapter structs to assert the public lifecycle contract. [Check](database-access-storage.go)
  Run: `./.ai-factory/rules/checkrules database-access-storage`
- [Keep dependency injection in bootstrap.go](dependency-injection-bootstrap.md) — Only root-level `bootstrap.go` imports the concrete Codex and Claude provider packages; provider options and shared CLI helpers remain available to providers. [Check](dependency-injection-bootstrap.go)
  Run: `./.ai-factory/rules/checkrules dependency-injection-bootstrap`
- [Centralize direct Git execution](git-execution-workspace.md) — In configured Go source, including tests, invoke or locate the configured Git executable only inside the configured Git execution boundary; other code uses its contract or adapter. [Check](git-execution-workspace.go)
  Run: `./.ai-factory/rules/checkrules git-execution-workspace`
- [Depend on consumer-owned interfaces](go-dependency-direction.md) — The `internal/agents` package does not import its `llm` provider subtree; providers may use agent contracts and shared helpers but not CLI commands. [Check](go-dependency-direction.go)
  Run: `./.ai-factory/rules/checkrules go-dependency-direction`
- [Go readability limits](go-readability-limits.md) — In Go under the configured paths, exclude tests, generated, vendored, and hidden metadata files; keep files at most 300 physical lines, named functions or methods at most 60 physical lines, and GoCognit complexity at most 15. [Check](go-readability-limits.go)
  Run: `./.ai-factory/rules/checkrules go-readability-limits`
- [Keep interface declaration files focused](interface-files.md) — Every Go file that declares a named interface, including tests, contains only imports, interfaces, and transitively used non-struct supporting types. [Check](interface-files.go)
  Run: `./.ai-factory/rules/checkrules interface-files`
- [Name interfaces by role](interface-type-names.md) — Named Go interfaces in the configured source roots, including tests, must not contain `Interface` in their names. [Check](interface-type-names.go)
  Run: `./.ai-factory/rules/checkrules interface-type-names`
- [Provider implementations satisfy consumer contracts](llm-adapter-port.md) — Keep provider SDK, endpoint, and CLI access under `internal/agents/llm` and assert `agents.AgentRunner` conformance. [Check](llm-adapter-port.go)
  Run: `./.ai-factory/rules/checkrules llm-adapter-port`
- [Organize packages by responsibility](package-organization.md) — Agents own RuleAgent and the Agent/AgentRunner contracts; llm owns options, Codex and Claude implement the runner contract, and runnercli is shared infrastructure. [Check](package-organization.go)
  Run: `./.ai-factory/rules/checkrules package-organization`
- [Co-locate struct methods and conformance assertions](struct-methods-and-contracts.md) — In configured Go source, including tests, put every named struct's receiver methods and explicit conformance assertions in its declaration file after the struct. [Check](struct-methods-and-contracts.go)
  Run: `./.ai-factory/rules/checkrules struct-methods-and-contracts`
