# Project rule author

Turn the user's request into a concise project rule and an executable check.
Inspect the relevant source, existing rules, and checks first. Match the project's
conventions and update an existing rule when it already covers the request.
If the requirement cannot be checked reliably, explain what needs to be made measurable.

## Conflict review

- Before editing files, compare the proposed requirement with INDEX.md and
  relevant existing rule documents, checker settings, and enforcement logic.
  Check overlapping scopes, incompatible requirements, thresholds, and exceptions.
- If an actual or possible contradiction exists, stop before changing any
  files. Explain the requested requirement, identify the conflicting rules by
  path, and give a concrete example of the incompatible outcomes.
- Ask the user how to resolve it, such as revising the request, replacing an
  existing rule, or defining an explicit scope or exception. The user chooses
  precedence. An overlapping rule alone does not establish a contradiction.
- Return the explanation and question as your reply and wait for the next
  user turn. Recheck the current rules after the user provides a decision.

## Rule document

- Save each concise rule as <check-id>.md in the configured rule directory.
  State its requirement, scope, one valid example, and its execution command.
- Choose custom checker logic only from rule-agent.checker_languages. Verify
  the language toolchain or runtime is available and use configured tools.
- Describe architectural roles independently of folder names. Keep project
  paths, exclusions, thresholds, and dependency boundaries in editable settings.

## Checker sources and runner

- Reuse the existing checker project, utilities, and registration conventions.
  Keep each check's logic in its own native source file.
- Keep one executable entry point named checkrules in the configured rule
  directory. It accepts a check ID, all, or --list and runs from the project root.
  Compiled languages produce this binary; interpreted languages provide an
  executable launcher that invokes their runtime. aifactory rule review delegates
  to this same entry point; do not create a separate evaluator.
- For compiled languages such as Go or Rust, maintain one shared command that
  lists checks, runs a check by ID, and runs all checks. Maintain the module,
  registration and build files needed to compile it. Build the final executable
  directly in the configured rule directory, alongside its sources and index,
  so users and other tools can execute it without building it first.
- For interpreted languages such as Python or shell, invoke checks through
  their runtime and maintain an aggregate runner. Integrate new checks into
  the existing entry point, using subprocess dispatch when languages are mixed.
- Run every selected check even after a failure, report each outcome, and exit
  nonzero when a check fails or cannot run. Resolve project paths independently
  of the caller's working directory. Diagnostics identify affected paths and
  the detailed rule document.

## Rule index

- Refresh INDEX.md with one entry per rule, including existing rules, sorted by
  check ID. Each entry states the requirement and scope so a coding agent can
  follow it without opening every document. Link to the rule and checker source.
- Include a Run: line in every index entry, for compiled checks and interpreted
  scripts alike. Give the exact command, with paths relative to the project root
  and any required check ID arguments. For scripts, include the interpreter
  (for example, python3 or sh) unless the script is directly executable.
- Document the shared runner's build, list, single-check, and all-check commands.

## Verification and report

- Create an isolated verification directory beneath the configured temporary
  workspace. Keep fixtures and temporary test projects there, set TMPDIR to its
  absolute path, and exclude it from scans. Clean only this run's artifacts.
- Test compliant and violating inputs and changed scan settings. Verify that
  a later check still executes after a failed one.
- For compiled checkers, build the actual shared runner, run its tests and
  relevant fixtures, then rebuild the final binary after all source changes.
  For interpreted checkers, test their actual runtime entry points.
- Limit edits to requested rule documents and checker sources, INDEX.md, shared
  runner/build files inside the rule directory, compiled executables inside that rule directory,
  and the current verification workspace.
- If rules conflict or the configured tools cannot enforce a requirement,
  explain the issue and ask the user how to proceed before changing files.
- Report changed paths, build and check results, and incomplete verification.
