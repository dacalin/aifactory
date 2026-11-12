package agents

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dacalin/aifactory/internal/config"
)

type recordingRunner struct {
	prompts []string
	fail    bool
}

func (runner *recordingRunner) Run(_ string, prompt string, onMessage func(string)) (string, error) {
	runner.prompts = append(runner.prompts, prompt)
	if runner.fail {
		runner.fail = false
		return "", errors.New("temporary provider failure")
	}
	if onMessage != nil {
		onMessage("reply")
	}
	return "reply", nil
}

func TestConstructorPreparesConfigurationBeforeConversation(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.WriteFile("factory.yaml", []byte("rules_dir: checks\nchecker_languages: [go]\n"), 0600); err != nil {
		t.Fatal(err)
	}
	runner := &recordingRunner{}
	cfg, err := config.Load(".")
	if err != nil {
		t.Fatal(err)
	}
	writeRulePrompt(t, cfg, "Follow this customized project prompt.")
	agent, err := NewRuleAgent(runner, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(runner.prompts) != 0 {
		t.Fatal("constructor sent a conversation turn")
	}
	if err := os.Remove("factory.yaml"); err != nil {
		t.Fatal(err)
	}
	for _, prompt := range []string{"Create a rule.", "Apply it everywhere."} {
		if _, err := agent.Converse(prompt, nil); err != nil {
			t.Fatal(err)
		}
	}
	if !strings.Contains(runner.prompts[0], "Follow this customized project prompt.") ||
		!strings.Contains(runner.prompts[0], `Configured rule directory: "checks"`) ||
		!strings.Contains(runner.prompts[0], "Create a rule.") ||
		runner.prompts[1] != "Apply it everywhere." {
		t.Fatalf("unexpected conversation prompts %q", runner.prompts)
	}
}

func TestConverseRetainsInstructionsUntilAFirstTurnSucceeds(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.WriteFile("factory.yaml", []byte("rules_dir: checks\n"), 0600); err != nil {
		t.Fatal(err)
	}
	runner := &recordingRunner{fail: true}
	cfg, err := config.Load(".")
	if err != nil {
		t.Fatal(err)
	}
	writeRulePrompt(t, cfg, "Follow this customized project prompt.")
	agent, err := NewRuleAgent(runner, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := agent.Converse("Create a rule.", nil); err == nil {
		t.Fatal("expected provider failure")
	}
	if _, err := agent.Converse("Create a rule.", nil); err != nil {
		t.Fatal(err)
	}
	if runner.prompts[0] != runner.prompts[1] {
		t.Fatal("retry lost the initial agent instructions")
	}
}

func TestConstructorRejectsMissingConfiguration(t *testing.T) {
	t.Chdir(t.TempDir())
	if _, err := config.Load("."); err == nil {
		t.Fatal("constructor accepted a project without factory.yaml")
	}
}

func writeRulePrompt(t *testing.T, cfg config.Factory, content string) {
	t.Helper()
	name := filepath.Join(cfg.ProjectDir, cfg.RulePromptPath())
	if err := os.MkdirAll(filepath.Dir(name), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(name, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestNewConversationReadsEditedPrompt(t *testing.T) {
	cfg := config.Factory{ProjectDir: t.TempDir(), Dir: "custom"}
	runner := &recordingRunner{}
	for _, instructions := range []string{"First instructions", "Edited instructions"} {
		writeRulePrompt(t, cfg, instructions)
		agent, err := NewRuleAgent(runner, cfg)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := agent.Converse("request", nil); err != nil {
			t.Fatal(err)
		}
		if !strings.HasPrefix(runner.prompts[len(runner.prompts)-1], instructions) {
			t.Fatal("edited prompt was not loaded")
		}
	}
}

func TestConstructorRejectsMissingOrEmptyPrompt(t *testing.T) {
	cfg := config.Factory{ProjectDir: t.TempDir(), Dir: ".ai-factory"}
	runner := &recordingRunner{}
	if _, err := NewRuleAgent(runner, cfg); err == nil || !strings.Contains(err.Error(), "aifactory init") {
		t.Fatalf("missing prompt: %v", err)
	}
	writeRulePrompt(t, cfg, " \n")
	if _, err := NewRuleAgent(runner, cfg); err == nil {
		t.Fatal("accepted empty prompt")
	}
	if len(runner.prompts) != 0 {
		t.Fatal("invalid prompt invoked provider")
	}
}
