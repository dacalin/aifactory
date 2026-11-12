# AI Factory

Turn project conventions into documented rules and executable checks. Use Codex or Claude Code to create rules, then run the checks locally, in a worktree, or from another tool.

## Install

Download your platform's archive from [Releases](https://github.com/dacalin/aifactory/releases), extract it, and put `aifactory` (`aifactory.exe` on Windows) on your `PATH`. Verify downloads against `SHA256SUMS`.

| Platform | Architectures |
| --- | --- |
| macOS | Intel (`amd64`), Apple Silicon (`arm64`) |
| Linux | x86 (`386`), x64 (`amd64`), ARMv7, ARM64 |
| Windows | x86 (`386`), x64 (`amd64`), ARM64 |

Install and sign in to your chosen **Codex CLI** or **Claude Code**. Creating or rebuilding Go checks also requires **Go 1.25+**. Other checker languages need their own runtime or compiler. Release binaries are unsigned.

With Go installed, you can also run `go install github.com/dacalin/aifactory@latest`.

## Use

Run from your project directory:

```sh
aifactory init
aifactory rule "Interfaces must live in separate files without structs or functions"
aifactory rule review
aifactory rule review interface-files
aifactory rule review --list
```

Rule creation streams progress and supports follow-up messages. Type `/exit` to finish. Review runs the shared checker directly, reports failures, and exits nonzero on failure. It uses the current checkout/worktree, including uncommitted changes; Go checkers are rebuilt first. The rule name `review` is reserved for this subcommand.

`init` creates missing files and preserves existing configuration and prompts. Customize `factory.yaml`:

```yaml
dir: .ai-factory/
rule-agent:
  agent: codex # or claude
  # model: your-model-id
  # effort: medium
  checker_languages: [go]
  tools:
    go_list: [go, list]
    go_run: [go, run]
```

Edit `.ai-factory/prompts/rule-agent.md` to customize the agent. Rules, checker sources, the shared executable, and `INDEX.md` live under `.ai-factory/rules` by default. Every index entry includes a command to run its check. Temporary verification files stay under `.ai-factory/.tmp`. Review uses those generated checkers; a fresh project needs its first rule before it can be reviewed.

## Develop

```sh
go test ./...
go -C .ai-factory/rules test ./...
go run . rule review
go run ./tools/release -version v0.1.0
```

Enable the shared pre-commit hook with `git config core.hooksPath .githooks`.

Release archives and checksums are written to `dist/`. Model-backed creation requires a configured CLI; automated tests use simulated providers. This repository's rules are project-specific; `init` gives other projects the editable prompt and configuration, without copying these rules.
