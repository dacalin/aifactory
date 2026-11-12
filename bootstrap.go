package main

import (
	"fmt"
	"os"

	"github.com/dacalin/aifactory/internal/agents"
	"github.com/dacalin/aifactory/internal/commands"
	"github.com/dacalin/aifactory/internal/config"
	"github.com/dacalin/aifactory/internal/llm"

	"github.com/spf13/cobra"
)

func bootstrap() *cobra.Command {
	return commands.NewRootCommand(newRuleAgent)
}

func newRuleAgent() (agents.Agent, error) {
	projectDir, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	cfg, err := config.Load(projectDir)
	if err != nil {
		return nil, err
	}
	var runner agents.AgentRunner
	switch cfg.RuleAgent.Agent {
	case "codex":
		runner = llm.NewCodexCLIRunner(cfg.RuleAgent.Model, cfg.RuleAgent.Effort)
	case "claude":
		runner = llm.NewClaudeCLIRunner(cfg.RuleAgent.Model, cfg.RuleAgent.Effort)
	default:
		return nil, fmt.Errorf("unsupported rule-agent.agent %q", cfg.RuleAgent.Agent)
	}
	return agents.NewRuleAgent(runner, cfg)
}
