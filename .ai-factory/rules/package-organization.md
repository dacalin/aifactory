# Organize agents and model implementations by responsibility

The configured agents package owns RuleAgent, Agent, and AgentRunner.
The configured llm package owns model-provider implementations. Bootstrap connects
those implementations to agents and commands.

Keep contract declarations in the agents package and concrete model access in
the llm package. Configure the package roots and required declarations at the top
of the checker. Retired ports/adapters package roots must contain no Go source.

For example, agents/agent_runner.go declares the runner contract and
llm/claude_cli_runner.go implements it.

Run ./.ai-factory/rules/checkrules package-organization from the project root.
