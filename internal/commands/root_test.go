package commands

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dacalin/aifactory/internal/agents"
	"github.com/dacalin/aifactory/internal/config"
)

type recordingRunner struct {
	prompts []string
}

func (runner *recordingRunner) Run(_ string, prompt string, onMessage func(string)) (string, error) {
	runner.prompts = append(runner.prompts, prompt)
	reply := fmt.Sprintf("Reply %d", len(runner.prompts))
	onMessage(reply)
	return reply, nil
}

func TestInitPreservesExistingConfiguration(t *testing.T) {
	t.Chdir(t.TempDir())
	runner := &recordingRunner{}
	factoryCalls := 0
	for _, expected := range []string{config.DefaultYAML, "rules_dir: custom/rules\n"} {
		command := NewRootCommand(func() (agents.Agent, error) {
			factoryCalls++
			cfg, err := config.Load(".")
			if err != nil {
				return nil, err
			}
			return agents.NewRuleAgent(runner, cfg)
		})
		command.SetArgs([]string{"init"})
		command.SetOut(&bytes.Buffer{})
		if err := command.Execute(); err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile("factory.yaml")
		if err != nil || string(data) != expected {
			t.Fatalf("factory.yaml = %q, %v", data, err)
		}
		if err := os.WriteFile("factory.yaml", []byte("rules_dir: custom/rules\n"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if factoryCalls != 0 || len(runner.prompts) != 0 {
		t.Fatal("init constructed or invoked the agent")
	}
}

func TestRuleCommandStreamsMultipleTurns(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.WriteFile("factory.yaml", []byte(config.DefaultYAML), 0600); err != nil {
		t.Fatal(err)
	}
	if err := initProject(&bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	runner := &recordingRunner{}
	var output bytes.Buffer
	command := NewRootCommand(func() (agents.Agent, error) {
		cfg, err := config.Load(".")
		if err != nil {
			return nil, err
		}
		return agents.NewRuleAgent(runner, cfg)
	})
	command.SetArgs([]string{"rule", "Create", "a", "rule"})
	command.SetIn(strings.NewReader("\nApply it to all paths.\n/exit\n"))
	command.SetOut(&output)
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	if len(runner.prompts) != 2 {
		t.Fatalf("expected two turns, got %d", len(runner.prompts))
	}
	if !strings.Contains(runner.prompts[0], "Create a rule") || runner.prompts[1] != "Apply it to all paths." {
		t.Fatalf("unexpected prompts %q", runner.prompts)
	}
	for _, reply := range []string{"Reply 1", "Reply 2"} {
		if strings.Count(output.String(), reply) != 1 {
			t.Fatalf("expected one streamed %q in %q", reply, output.String())
		}
	}
}

func TestHelpAndInvalidArgumentsDoNotInvokeTheAgent(t *testing.T) {
	for _, args := range [][]string{
		{"--help"}, {"init", "--help"}, {"rule", "--help"}, {}, {"init", "extra"}, {"--unknown"}, {" "}, {"rule"}, {"rule", " "}, {"rule", "--unknown"}, {"Create a rule"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			runner := &recordingRunner{}
			factoryCalls := 0
			command := NewRootCommand(func() (agents.Agent, error) {
				factoryCalls++
				cfg, err := config.Load(".")
				if err != nil {
					return nil, err
				}
				return agents.NewRuleAgent(runner, cfg)
			})
			command.SetArgs(args)
			command.SetOut(&bytes.Buffer{})
			command.SetErr(&bytes.Buffer{})
			err := command.Execute()
			help := len(args) == 0 || args[len(args)-1] == "--help"
			if help && err != nil || !help && err == nil {
				t.Fatalf("Execute(%q) returned %v", args, err)
			}
			if factoryCalls != 0 || len(runner.prompts) != 0 {
				t.Fatal("argument handling constructed or invoked the agent")
			}
		})
	}
}

func TestInitPreservesCustomConfigurationAndPrompt(t *testing.T) {
	t.Chdir(t.TempDir())
	settings := "# keep this comment\ndir: custom\nrule-agent:\n  dir: checks\n  agent: codex\n  model: test-model\n  effort: medium\n"
	if err := os.WriteFile("factory.yaml", []byte(settings), 0600); err != nil {
		t.Fatal(err)
	}
	name := filepath.Join("custom", "prompts", "rule-agent.md")
	for _, expected := range []string{agents.DefaultRulePrompt, "User-edited prompt"} {
		if err := initProject(&bytes.Buffer{}); err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(name)
		if err != nil || string(data) != expected {
			t.Fatalf("prompt = %q, %v", data, err)
		}
		data, err = os.ReadFile("factory.yaml")
		if err != nil || string(data) != settings {
			t.Fatalf("configuration changed: %q, %v", data, err)
		}
		if err := os.WriteFile(name, []byte("User-edited prompt"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Remove(name); err != nil {
		t.Fatal(err)
	}
	if err := initProject(&bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(name)
	if err != nil || string(data) != agents.DefaultRulePrompt {
		t.Fatalf("missing prompt not restored: %q, %v", data, err)
	}
}

func TestInitPreservesInvalidExistingYAML(t *testing.T) {
	t.Chdir(t.TempDir())
	const invalid = "dir: [invalid"
	if err := os.WriteFile("factory.yaml", []byte(invalid), 0600); err != nil {
		t.Fatal(err)
	}
	if err := initProject(&bytes.Buffer{}); err == nil {
		t.Fatal("accepted invalid configuration")
	}
	data, err := os.ReadFile("factory.yaml")
	if err != nil || string(data) != invalid {
		t.Fatalf("existing YAML replaced: %q, %v", data, err)
	}
}
