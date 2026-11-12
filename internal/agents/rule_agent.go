package agents

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dacalin/aifactory/internal/config"

	"gopkg.in/yaml.v3"
)

type RuleAgent struct {
	runner       AgentRunner
	projectDir   string
	instructions string
	started      bool
}

var _ Agent = (*RuleAgent)(nil)

func NewRuleAgent(runner AgentRunner, cfg config.Factory) (*RuleAgent, error) {
	name := filepath.Join(cfg.ProjectDir, cfg.RulePromptPath())
	instructions, err := os.ReadFile(name)
	if err != nil {
		return nil, fmt.Errorf("read rule-agent prompt %s (run aifactory init to create it): %w", name, err)
	}
	if strings.TrimSpace(string(instructions)) == "" {
		return nil, fmt.Errorf("rule-agent prompt %s must not be empty", name)
	}
	settings, err := yaml.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("format factory.yaml settings: %w", err)
	}
	return &RuleAgent{
		runner:       runner,
		projectDir:   cfg.ProjectDir,
		instructions: fmt.Sprintf("%s\n\nProject configuration (YAML, paths relative to the project root):\n%s\nConfigured rule directory: %q\nConfigured temporary workspace: %q\n\nUser request:\n", instructions, settings, cfg.RulesDir(), cfg.TemporaryDir()),
	}, nil
}

func (agent *RuleAgent) Converse(prompt string, onMessage func(string)) (string, error) {
	input := prompt
	if !agent.started {
		input = agent.instructions + prompt + "\n"
	}
	reply, err := agent.runner.Run(agent.projectDir, input, onMessage)
	if err == nil {
		agent.started = true
	}
	return reply, err
}
