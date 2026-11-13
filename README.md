# AI Factory

Turn project conventions into documented rules and executable checks. Use Codex or Claude Code to create rules, then run the checks locally, in a worktree, or from another tool.

## Install

AI Factory supports macOS and Linux. Choose one of these options. Prebuilt downloads do not require Go to install the app. Creating or rebuilding Go rule checkers requires **Go 1.25+**; other checker languages need their own runtime or compiler. To create rules, also install and sign in to **Codex CLI** or **Claude Code**.

The repository is currently private: installation requires a GitHub account with access. Sign in to GitHub in your browser for manual downloads, or run `gh auth login` with the GitHub CLI for terminal downloads.

### Option 1: Download a binary

Download an archive from [Releases](https://github.com/dacalin/aifactory/releases/latest). Choose the operating system and architecture below; `darwin` means macOS.

| Platform | Processor | Archive suffix |
| --- | --- | --- |
| macOS | Intel | `darwin_amd64.tar.gz` |
| macOS | Apple Silicon | `darwin_arm64.tar.gz` |
| Linux | x64 | `linux_amd64.tar.gz` |
| Linux | ARM64 | `linux_arm64.tar.gz` |
| Linux | ARMv7 | `linux_armv7.tar.gz` |
| Linux | 32-bit x86 | `linux_386.tar.gz` |

On macOS/Linux, `uname -m` reports the architecture: `x86_64` means `amd64`, `arm64`/`aarch64` means `arm64`, and `armv7l` means `armv7`. Release binaries are unsigned.

**macOS and Linux**

The terminal example uses the authenticated GitHub CLI (`gh`). You can also download the same files in your browser. This example installs v0.1.1 for Apple Silicon. Change `OS` and `ARCH` to match the table; for a typical Linux PC, use `OS=linux` and `ARCH=amd64`. Change `VERSION` when installing a newer release.

```sh
(
  set -eu
  VERSION=v0.1.1
  OS=darwin
  ARCH=arm64
  ARCHIVE="aifactory_${VERSION}_${OS}_${ARCH}.tar.gz"
  WORKDIR=$(mktemp -d)
  cd "$WORKDIR"

  gh release download "$VERSION" --repo dacalin/aifactory \
    --pattern "$ARCHIVE" --pattern SHA256SUMS
  if [ "$OS" = darwin ]; then
    shasum -a 256 --check --ignore-missing SHA256SUMS
  else
    sha256sum --check --ignore-missing SHA256SUMS
  fi

  tar -xzf "$ARCHIVE"
  mkdir -p "$HOME/.local/bin"
  install -m 755 aifactory "$HOME/.local/bin/aifactory"
)
export PATH="$HOME/.local/bin:$PATH"
aifactory --version
```

Add `export PATH="$HOME/.local/bin:$PATH"` to `~/.zshrc` for zsh or `~/.bashrc` for bash to keep the command available in new terminals. No administrator access is needed.

### Option 2: Install with Go

With Go 1.25+ installed, configure Git authentication for the private repository. If you use the GitHub CLI, run `gh auth setup-git` after signing in. Then add the repository to `GOPRIVATE` so Go fetches it directly rather than using the public module proxy.

macOS/Linux:

```sh
export GOPRIVATE="${GOPRIVATE:+$GOPRIVATE,}github.com/dacalin/aifactory"
```

Then run this from any directory:

```sh
go install github.com/dacalin/aifactory@latest
```

For a specific release, replace `@latest` with `@v0.1.1`. Go installs the executable in `GOBIN` if set; otherwise it uses the first `GOPATH` entry's `bin` directory, normally `~/go/bin`. Inspect your settings with `go env GOBIN GOPATH` and add the installation directory to `PATH`.

For the default Go location on macOS/Linux:

```sh
export PATH="$HOME/go/bin:$PATH"
aifactory --help
```

Persist that export in your shell profile as described above. Source installs may report `dev` as their version.

### Option 3: Build or run from source

Requires Git and Go 1.25+, with Git authentication configured for this repository (for example, `gh auth setup-git`).

```sh
git clone https://github.com/dacalin/aifactory.git
cd aifactory
go build -trimpath -o aifactory .
./aifactory --help
```

Copy the built executable to a directory on `PATH` to use it from other projects.

To run without installing, use `go run . --help` from the cloned repository. To update, download and replace the binary, rerun `go install github.com/dacalin/aifactory@latest`, or run `git pull --ff-only` and rebuild, depending on your installation method.

## Use

| Command | Purpose |
| --- | --- |
| `aifactory init` | Create missing YAML and editable prompts; preserve existing files. |
| `aifactory rule "description"` | Create or update a rule through a conversation. |
| `aifactory rule review [check-id\|all]` | Run one check or all checks (the default) against the current project/worktree. |
| `aifactory rule review --list` | List available check IDs. |

Run from your project directory:

```sh
aifactory init
aifactory rule "Interfaces must live in separate files without structs or functions"
aifactory rule review
aifactory rule review interface-files
aifactory rule review --list
```

Use `aifactory --help`, `aifactory rule --help`, or `aifactory rule review --help` for command details and examples.

Rule creation streams progress and supports follow-up messages. Type `/exit` to finish. Review runs the shared checker directly, reports failures, and exits nonzero on failure. It uses the current checkout/worktree, including uncommitted changes; the optional `checks.build` command runs first. The rule name `review` is reserved for this subcommand.

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

RuleAgent configures one shared checker through top-level `checks` settings. Commands are argument lists, run from the configured rule directory, and execute without shell parsing. Review appends `all`, a check ID, or `--list` to `checks.run`. The project root is available as `AIFACTORY_PROJECT_DIR`. The app makes no assumptions about language, filenames, extensions, or how to compile checks.

For Python (no build step):

```yaml
checks:
  run: [python3, checkrules.py]
```

For JavaScript, use `run: [node, checkrules.js]`. For a compiled Go checker:

```yaml
checks:
  build: [go, build, -o, checkrules, .]
  run: [./checkrules]
```

Existing projects without `checks.run` need it added before using review; ask RuleAgent to configure the runner or add the commands manually.

Edit `.ai-factory/prompts/rule-agent.md` to customize the agent. Rules, checker sources, the shared checker entry point, and `INDEX.md` live under `.ai-factory/rules` by default. Every index entry includes a command to run its check. Temporary verification files stay under `.ai-factory/.tmp`. Review uses those generated checkers; a fresh project needs its first rule before it can be reviewed.

## Develop

```sh
go test ./...
go -C .ai-factory/rules test ./...
go run . rule review
make build
make release VERSION=v0.1.2
```

Enable the shared pre-commit hook with `git config core.hooksPath .githooks`.

The Makefile invokes Go directly. `make build` creates the local executable in `bin/`; `make release VERSION=v0.1.2` builds all six macOS/Linux targets and writes binaries, archives, and `SHA256SUMS` under `dist/v0.1.2/`. Release packaging requires Make, Go, `tar`, and either `sha256sum` or `shasum`. `make test` runs the application and checker tests. Model-backed creation requires a configured CLI; automated tests use simulated providers. This repository's rules are project-specific; `init` gives other projects the editable prompt and configuration, without copying these rules.
