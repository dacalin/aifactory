# Depend on consumer-owned interfaces

Agents own their contracts and depend on injected runners. Model implementations
may import those contracts; agents must not import model implementations.
Model implementations must not import CLI commands. Bootstrap connects the
implementations to their consumers.

Configure forbidden source/target import pairs in the checker settings. Pairs
cover descendant packages. For example, internal/agents internal/llm blocks
agents from importing concrete model implementations.

Run ./.ai-factory/rules/checkrules go-dependency-direction from the project root.
The checker inspects resolved direct imports for the active Go build.
