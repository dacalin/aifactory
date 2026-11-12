# Project rules

The ready-to-run binary lives alongside the rules. Rebuild it from the project
root after changing checker sources.

```sh
go -C .ai-factory/rules build -o checkrules .
```

The runner reads artifact and rule paths from `factory.yaml`. Use `--root` to
select a project, or `--data-dir` and `--rules-dir` to override these paths.

Use the CLI from this checkout or worktree (including subdirectories). It runs
this same entry point and rebuilds Go checks from the current sources first.
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

- [Keep declarations focused](adapter-port-one-struct.md) — Keep one named implementation struct or agent interface per declaration file, with matching filenames; allow helper files. [Check](adapter-port-one-struct.go)
  Run: `./.ai-factory/rules/checkrules adapter-port-one-struct`
- [Construct runner-backed agents through injection](agent-constructor-injection.md) — In configured agent Go code, name runner-backed structs `{Name}Agent`; make runner calls only from their methods and inject/store the runner through `New{Name}Agent`. [Check](agent-constructor-injection.go)
  Run: `./.ai-factory/rules/checkrules agent-constructor-injection`
- [Make agents conform to the conversation interface](agent-converse-interface.md) — In configured agent Go code, give every `{Name}Agent` the shared `Converse(prompt string, onMessage func(string)) (string, error)` method and an explicit `Agent` conformance assertion. [Check](agent-converse-interface.go)
  Run: `./.ai-factory/rules/checkrules agent-converse-interface`
- [Separate CLI command responsibilities](cli-command-separation.md) — Use Cobra; keep root execution and error reporting in `main.go`, dependency wiring in `bootstrap.go`, and each configured command or conversation handler in its configured command file. [Check](cli-command-separation.go)
  Run: `./.ai-factory/rules/checkrules cli-command-separation`
- [Keep dependency injection in bootstrap.go](dependency-injection-bootstrap.md) — In configured Go source, import project-local concrete implementation packages only from root-level `bootstrap.go`; keep both root command files in `package main`. [Check](dependency-injection-bootstrap.go)
  Run: `./.ai-factory/rules/checkrules dependency-injection-bootstrap`
- [Depend on consumer-owned interfaces](go-dependency-direction.md) — Agents depend on their runner contracts; model implementations may import agents and must not import CLI commands. [Check](go-dependency-direction.go)
  Run: `./.ai-factory/rules/checkrules go-dependency-direction`
- [Go readability limits](go-readability-limits.md) — In Go under the configured paths, exclude tests, generated, vendored, and hidden metadata files; keep files at most 300 physical lines, named functions or methods at most 60 physical lines, and GoCognit complexity at most 15. [Check](go-readability-limits.go)
  Run: `./.ai-factory/rules/checkrules go-readability-limits`
- [Keep interface declaration files focused](interface-files.md) — Every Go file that declares a named interface, including tests, contains only imports, interfaces, and transitively used non-struct supporting types. [Check](interface-files.go)
  Run: `./.ai-factory/rules/checkrules interface-files`
- [Name interfaces by role](interface-type-names.md) — Named Go interfaces in the configured source roots, including tests, must not contain `Interface` in their names. [Check](interface-type-names.go)
  Run: `./.ai-factory/rules/checkrules interface-type-names`
- [Model implementations satisfy consumer contracts](llm-adapter-port.md) — Keep model access in llm and assert conformance to interfaces owned by agents. [Check](llm-adapter-port.go)
  Run: `./.ai-factory/rules/checkrules llm-adapter-port`
- [Organize packages by responsibility](package-organization.md) — Agents own RuleAgent and the Agent/AgentRunner contracts; llm owns provider implementations; bootstrap wires consumers. [Check](package-organization.go)
  Run: `./.ai-factory/rules/checkrules package-organization`
