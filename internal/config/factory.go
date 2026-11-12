package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const DefaultYAML = `# Prompts are read from dir/prompts; init creates missing default prompts.
dir: .ai-factory/

rule-agent:
  # Rule files go under dir/rules by default; this path is relative to dir.
  # dir: rules
  agent: codex # codex or claude

  # Optional model ID for the selected agent; omit to use its CLI default.
  # model: your-model-id

  # Optional reasoning effort; omit to use the CLI default.
  # Supported levels depend on the selected provider and model.
  # effort: high

  # Languages the agent may use for custom checker logic.
  # Choose the simplest suitable option and verify its runtime is available.
  # Compiled checks share a binary in the rule directory; interpreted checks use their runtime.
  checker_languages: [go]

  # Commands the rule agent can use. Each entry lists an executable and arguments.
  tools:
    go_list: [go, list]
    go_run: [go, run]
    gocognit: [go, run, "github.com/uudashr/gocognit/cmd/gocognit@v1.2.2"]
`

type Factory struct {
	Dir        string          `yaml:"dir"`
	RuleAgent  RuleAgentConfig `yaml:"rule-agent"`
	ProjectDir string          `yaml:"-"`
}

type RuleAgentConfig struct {
	Dir              string              `yaml:"dir"`
	Agent            string              `yaml:"agent"`
	Model            string              `yaml:"model,omitempty"`
	Effort           string              `yaml:"effort,omitempty"`
	CheckerLanguages []string            `yaml:"checker_languages"`
	Tools            map[string][]string `yaml:"tools,omitempty"`
}

func Load(projectDir string) (Factory, error) {
	dir, err := filepath.Abs(projectDir)
	if err != nil {
		return Factory{}, fmt.Errorf("resolve project directory: %w", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "factory.yaml"))
	if err != nil {
		return Factory{}, fmt.Errorf("read factory.yaml: %w", err)
	}
	var raw struct {
		Dir              string              `yaml:"dir"`
		RuleAgent        *RuleAgentConfig    `yaml:"rule-agent"`
		RulesDir         *string             `yaml:"rules_dir"`
		CheckerLanguages []string            `yaml:"checker_languages"`
		Tools            map[string][]string `yaml:"tools"`
	}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return Factory{}, fmt.Errorf("parse factory.yaml: %w", err)
	}
	if raw.RulesDir != nil {
		if raw.Dir != "" || raw.RuleAgent != nil {
			return Factory{}, fmt.Errorf("factory.yaml: use either rules_dir or dir with rule-agent")
		}
		return legacyFactory(dir, *raw.RulesDir, raw.CheckerLanguages, raw.Tools)
	}
	cfg := Factory{ProjectDir: dir, Dir: raw.Dir}
	if raw.RuleAgent != nil {
		cfg.RuleAgent = *raw.RuleAgent
	}
	return cfg.normalized()
}

func legacyFactory(projectDir, rulesDir string, languages []string, tools map[string][]string) (Factory, error) {
	rulesDir = filepath.Clean(strings.TrimSpace(rulesDir))
	if rulesDir == "." || !filepath.IsLocal(rulesDir) {
		return Factory{}, fmt.Errorf("factory.yaml: rules_dir must be a project-relative subdirectory")
	}
	if languages == nil {
		languages = []string{"shell", "python", "go"}
	}
	cfg := Factory{
		ProjectDir: projectDir,
		Dir:        filepath.Dir(rulesDir),
		RuleAgent:  RuleAgentConfig{Dir: filepath.Base(rulesDir), Agent: "codex", CheckerLanguages: languages, Tools: tools},
	}
	return cfg.normalized()
}

func (cfg Factory) normalized() (Factory, error) {
	if cfg.Dir == "" {
		cfg.Dir = ".ai-factory"
	}
	if cfg.RuleAgent.Dir == "" {
		cfg.RuleAgent.Dir = "rules"
	}
	cfg.Dir = filepath.Clean(cfg.Dir)
	cfg.RuleAgent.Dir = filepath.Clean(cfg.RuleAgent.Dir)
	if !filepath.IsLocal(cfg.Dir) || !filepath.IsLocal(cfg.RuleAgent.Dir) || cfg.RulesDir() == "." {
		return Factory{}, fmt.Errorf("factory.yaml: dir and rule-agent.dir must resolve to a project-relative rule directory")
	}
	cfg.RuleAgent.Model = strings.TrimSpace(cfg.RuleAgent.Model)
	cfg.RuleAgent.Effort = strings.ToLower(strings.TrimSpace(cfg.RuleAgent.Effort))
	cfg.RuleAgent.Agent = strings.ToLower(strings.TrimSpace(cfg.RuleAgent.Agent))
	if cfg.RuleAgent.Agent == "" {
		cfg.RuleAgent.Agent = "codex"
	}
	if cfg.RuleAgent.Agent != "codex" && cfg.RuleAgent.Agent != "claude" {
		return Factory{}, fmt.Errorf("factory.yaml: rule-agent.agent must be codex or claude")
	}
	if cfg.RuleAgent.CheckerLanguages == nil {
		cfg.RuleAgent.CheckerLanguages = []string{"go"}
	}
	if len(cfg.RuleAgent.CheckerLanguages) == 0 {
		return Factory{}, fmt.Errorf("factory.yaml: rule-agent.checker_languages must contain at least one language")
	}
	for _, language := range cfg.RuleAgent.CheckerLanguages {
		if strings.TrimSpace(language) == "" {
			return Factory{}, fmt.Errorf("factory.yaml: rule-agent.checker_languages must contain non-empty language names")
		}
	}
	return cfg, nil
}

func (cfg Factory) RulesDir() string {
	return filepath.Join(cfg.Dir, cfg.RuleAgent.Dir)
}

func (cfg Factory) TemporaryDir() string {
	return filepath.Join(cfg.Dir, ".tmp")
}

func (cfg Factory) RulePromptPath() string {
	return filepath.Join(cfg.Dir, "prompts", "rule-agent.md")
}
