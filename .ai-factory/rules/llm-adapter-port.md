# Model implementations satisfy consumer-owned contracts

Keep LLM SDK, endpoint, and CLI calls in the configured llm package.
Agents own the interfaces they consume. Each model implementation imports
that agents package and declares compile-time runner conformance.

For example, var _ agents.AgentRunner = (*CodexCLIRunner)(nil)
checks a model implementation against its consumer's contract.

Set source paths, the interface-owning package, implementation root, and exclusions
in the checker's settings. Run ./.ai-factory/rules/checkrules llm-adapter-port from the
project root. It checks recognized provider access and runs Go tests to compile
the assertions. Extend provider detection when introducing an unrecognized SDK,
CLI, or endpoint proxy.
