package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadConfiguration(t *testing.T) {
	cases := []struct {
		name, yaml, rules, provider string
		languages                   []string
		invalid                     bool
	}{
		{"default", DefaultYAML, ".ai-factory/rules", "codex", []string{"go"}, false},
		{"custom", "dir: policy\nrule-agent:\n  dir: quality\n  agent: claude\n  checker_languages: [python]\n", "policy/quality", "claude", []string{"python"}, false},
		{"legacy", "rules_dir: checks\nchecker_languages: [go]\n", "checks", "codex", []string{"go"}, false},
		{"legacy defaults", "rules_dir: custom/checks\n", "custom/checks", "codex", []string{"shell", "python", "go"}, false},
		{"provider", "rule-agent: {agent: unknown}\n", "", "", nil, true},
		{"empty languages", "rule-agent: {checker_languages: []}\n", "", "", nil, true},
		{"scalar languages", "rule-agent: {checker_languages: go}\n", "", "", nil, true},
		{"escape root", "dir: ../outside\n", "", "", nil, true},
		{"escape rules", "rule-agent: {dir: ../outside}\n", "", "", nil, true},
		{"mixed", "rules_dir: checks\nrule-agent: {agent: claude}\n", "", "", nil, true},
		{"invalid tools", "rule-agent: {tools: {go_list: go list}}\n", "", "", nil, true},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "factory.yaml"), []byte(test.yaml), 0600); err != nil {
				t.Fatal(err)
			}
			cfg, err := Load(dir)
			if test.invalid {
				if err == nil {
					t.Fatal("accepted invalid configuration")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if cfg.RulesDir() != filepath.FromSlash(test.rules) || cfg.RuleAgent.Agent != test.provider || !reflect.DeepEqual(cfg.RuleAgent.CheckerLanguages, test.languages) {
				t.Fatalf("unexpected normalized config %+v", cfg)
			}
			if cfg.ProjectDir != dir {
				t.Fatalf("project directory = %q", cfg.ProjectDir)
			}
		})
	}
}

func TestToolCommandsStayArgumentLists(t *testing.T) {
	dir := t.TempDir()
	data := "dir: policy\nrule-agent:\n  model: ' test-model '\n  effort: ' HIGH '\n  tools:\n    formatter: [python3, 'tools/format files.py', --check]\n"
	if err := os.WriteFile(filepath.Join(dir, "factory.yaml"), []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.RuleAgent.Model != "test-model" {
		t.Fatalf("model = %q", cfg.RuleAgent.Model)
	}
	if cfg.RuleAgent.Effort != "high" {
		t.Fatalf("effort = %q", cfg.RuleAgent.Effort)
	}
	expected := []string{"python3", "tools/format files.py", "--check"}
	if !reflect.DeepEqual(cfg.RuleAgent.Tools["formatter"], expected) {
		t.Fatalf("unexpected command %q", cfg.RuleAgent.Tools["formatter"])
	}
}
