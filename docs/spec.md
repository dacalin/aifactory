# AI Factory implemented specification

Each scenario describes behavior available in this checkout. See [architecture.md](architecture.md) for package boundaries and [dependencies.yaml](dependencies.yaml) for runtime requirements.

## Local server

- **When** the user runs `aifactory serve` from any directory, **then** AI Factory loads its global settings and SQLite database under `~/.ai-factory/`, serves the embedded UI, and prints the local URL. The default is `http://127.0.0.1:8787`.
- **When** the settings file is absent, **then** AI Factory writes the default `config.yaml`. A configured listen address must use `127.0.0.1` or `localhost` and an explicit valid port.
- **When** the user starts a second instance with the same data directory, **then** the server lock prevents that instance from starting.
- **When** the server starts, **then** it attempts to open the default browser unless `--no-browser` is set. Failure to open the browser is reported while the server continues running.
- **When** a request uses a non-loopback Host header or a nonmatching browser Origin, **then** the server rejects it with HTTP 403.
- **When** the server receives an interrupt or termination signal, **then** it shuts down HTTP, closes the database, and releases the lock. If graceful HTTP shutdown times out, it closes the server.

## Projects and folders

- **When** the user opens the UI, **then** it lists projects registered in the local database. Selecting a project shows its path, default branch, available branches, and tasks.
- **When** the user selects a folder through the native picker, **then** the server returns its path and the UI fills the corresponding field. Cancelling leaves the field as it was. If the desktop picker is unavailable, the UI keeps the existing input and explains how to paste a path. Both forms accept a pasted path.
- **When** the user creates a project, **then** AI Factory validates that the name is a single folder name, initializes Git at the selected location and name, and records the project. An existing empty Git repository can be reused. Other nonempty destinations are rejected.
- **When** the user opens an existing directory inside a Git working tree, **then** AI Factory resolves and registers the repository root. Opening a path that resolves to an already registered root returns the existing project.
- **When** a project is registered or an older project without a default branch is loaded, **then** AI Factory detects and stores a default. It prefers `origin/HEAD`, then `main`, `master`, `origin/main`, or `origin/master`, then an available current branch and finally a listed branch. A new repository can display its initial branch before its first commit.
- **When** a user saves a different default branch, **then** AI Factory validates that it exists and resolves to a commit, and stores it for future tasks.

## Tasks and Git workspaces

- **When** a Git hook supplies its own index via `GIT_INDEX_FILE`, **then** workspace commands use the index of the target worktree.

- **When** a user creates a task, **then** AI Factory requires a title and a base branch that resolves to a commit. The description is optional. The task is assigned `backlog` status and saved in SQLite.
- **When** the user opens the task form, **then** it preselects the project's default branch and allows a different branch to be selected. An API caller can omit the branch and inherit the project default.
- **When** the project default changes, **then** saved tasks keep their own branch. The project detail page shows the saved title, optional description, status, and base branch for each task.
- **When** application code calls the workspace manager to create a workspace, **then** it creates a Git worktree under the supplied workspace root at `<project-id>/<task-id>`. A new workspace uses a separate `aifactory/<project-id>/<task-id>` branch based on the task branch, with the project default as a fallback if the supplied task has no branch.
- **When** the workspace path already exists, **then** the manager reuses it only if it is a worktree for the same repository. Removal checks that the path resolves inside the managed workspace directory before invoking Git.
- **When** a task is created through the UI or HTTP API, **then** the current release only records it. The server has no task executor, task state transitions, automatic worktree creation, or PR creation. The saved branch is available to those future flows.

- **When** application code calls `MergeBranch`, **then** Git merges the named local branch into the current worktree branch.
- **When** application code calls `PushBranch`, **then** Git pushes the named local branch to the same branch on a configured remote using a normal push.

### HTTP routes

| When a local client calls | Then the server |
| --- | --- |
| `GET /api/health` | Returns `{"status":"ok"}`. |
| `GET /api/projects` | Lists registered projects. |
| `POST /api/projects` | Creates from `operation: create`, `name`, and `parent_path`, or opens from `operation: open` and `path`. Returns 201 for a new registration and 200 for an existing one. |
| `GET /api/projects/{id}` | Loads the registered project, or returns 404. |
| `PATCH /api/projects/{id}` | Validates and saves `base_branch`. |
| `GET /api/projects/{id}/branches` | Returns a JSON object containing the `branches` list. |
| `GET /api/projects/{id}/tasks` | Lists the project's tasks. |
| `POST /api/projects/{id}/tasks` | Records a task using `title`, optional `description`, and optional `base_branch`. Returns 201 on success. |
| `POST /api/folder-selection` | Opens the native picker for `operation: create` or `open`. Returns `path` or `{"cancelled":true}`. |

## Rule setup and authoring

- **When** the user runs `aifactory init`, **then** it creates missing `factory.yaml` and `<dir>/prompts/rule-agent.md` in the current project. Existing configuration and edited prompts are preserved. Invalid existing YAML is reported.
- **When** the project uses the default paths, **then** prompts are under `.ai-factory/prompts` and rules are under `.ai-factory/rules`. Custom `dir` and `rule-agent.dir` values must resolve to a project-relative rules directory. A legacy `rules_dir` configuration is accepted separately from the newer layout.
- **When** the user runs `aifactory rule "<request>"` from the project root, **then** the command loads `factory.yaml` and the editable prompt. The first turn receives the instructions and project configuration. Codex is the default provider. Claude Code is selectable, and both accept an optional model and reasoning effort.
- **When** the provider starts a turn, **then** it runs from the project root so its own project and MCP configuration can be discovered. AI Factory includes `checker_languages` and `tools` from `factory.yaml` in the initial prompt for the agent.
- **When** AI Factory starts a write-restricted provider session, **then** it sets `CODEX_HOME` or `CLAUDE_CONFIG_DIR` to `<rules>/.tmp/home/<provider-session>/` and `TMPDIR` and `GOTMPDIR` to `<rules>/.tmp/work/<provider-session>/`. The Go build cache lives in the latter directory. Follow-up turns reuse both directories, and Git ignores the runtime tree.
- **When** an interactive rule conversation returns after `/exit`, end of input, or a turn error, **then** AI Factory removes that session's provider home and scratch directory. The agent is directed to use `TMPDIR` for temporary files.
- **When** a selected provider emits progress or replies, **then** AI Factory streams them to the terminal. A nonempty follow-up resumes the provider session. Entering `/exit` or reaching end of input ends the conversation.
- **When** the caller cancels an active turn or its deadline expires, **then** the runner returns the context error and terminates the provider's process group. During a CLI turn, Ctrl+C and SIGTERM cancel the turn. File edits completed before cancellation can remain.
- **When** a rule request requires a change to the root `factory.yaml`, **then** the agent is instructed to propose the settings for the user to apply. Its writable directory is the configured rules directory.

## Rule review and write boundary

- **When** the user runs `aifactory rule review` from a project or one of its subdirectories, **then** AI Factory finds `factory.yaml` in the current checkout or worktree, including uncommitted changes. It stops searching at that checkout's Git boundary.
- **When** review is called without an argument, with one check ID, or with `--list`, **then** the CLI passes `all`, the supplied ID, or `--list` to the configured shared checker respectively.
- **When** `checks.build` is configured, **then** it runs first. Both commands run from the rules directory as argument arrays without shell parsing and receive `AIFACTORY_PROJECT_DIR`. Interpreted checkers can run with only `checks.run`.
- **When** configuration is missing, the build fails, or the checker exits nonzero, **then** review returns an error and exits nonzero. Review executes the checker directly and does not start an LLM.
- **When** a rule authoring or managed review command starts, **then** AI Factory applies an OS sandbox to the process. The rules directory is writable and the rest of the repository remains readable. Rule runtime files and build cache live below the rules directory in `.tmp`; review removes its per-run temporary directory on exit.
- **When** a write boundary points through a symbolic link, includes Git metadata, or cannot be applied by the OS, **then** AI Factory refuses to start the protected process. macOS uses `sandbox-exec`; Linux uses `bwrap` with user namespaces. Connected services enforce their own permissions.
- **When** contributors need to see the checks configured for AI Factory, **then** the [rule index](../.ai-factory/rules/INDEX.md) lists each rule, its checker source, and how to run it. Those checks are project-specific; `init` does not copy them into another project.
- **When** a contributor enables the shared Git hooks, **then** the pre-commit hook builds and invokes this repository's checker directly against the working tree.

## Build and release

- **When** the user runs `aifactory --version`, **then** it prints the version embedded in the binary. Source builds use `dev` by default.
- **When** a developer runs `go build` from a checkout containing `web/dist`, **then** the resulting binary includes the web UI. `make build` rebuilds those assets before compiling `bin/aifactory`.
- **When** a developer runs `make test`, **then** it rebuilds the UI and runs the application and checker module tests.
- **When** a developer runs `make release VERSION=vX.Y.Z`, **then** it builds the configured macOS and Linux binaries, packages them as archives, and writes `SHA256SUMS`.
