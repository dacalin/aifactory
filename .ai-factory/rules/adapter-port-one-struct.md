# Keep implementation and contract declarations focused

In configured model implementation paths, keep at most one top-level struct in
each Go file. In configured agent contract paths, keep at most one interface per
file. Name each declaration's file in snake case. Files containing only helper
functions are allowed, and test files are excluded.

For example, AgentRunner belongs in agent_runner.go and
ClaudeCLIRunner belongs in claude_cli_runner.go.

Set implementation paths, interface paths, and exclusions in the checker's settings.
Run ./.ai-factory/rules/checkrules adapter-port-one-struct from the project root.
