package llm

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/dacalin/aifactory/internal/agents"
)

func TestRunnerModelAndEffortSelection(t *testing.T) {
	for _, provider := range []struct {
		name      string
		newRunner func(string, string) agents.AgentRunner
		events    string
	}{
		{"codex", func(model, effort string) agents.AgentRunner { return NewCodexCLIRunner(model, effort) }, `{"type":"thread.started","thread_id":"model-test"}
{"type":"item.completed","item":{"type":"agent_message","text":"done"}}
{"type":"turn.completed"}`},
		{"claude", func(model, effort string) agents.AgentRunner { return NewClaudeCLIRunner(model, effort) }, `{"type":"system","session_id":"model-test"}
{"type":"result","subtype":"success","session_id":"model-test","result":"done","is_error":false}`},
	} {
		for _, settings := range []struct{ model, effort string }{{"", ""}, {" test-model ", " HIGH "}, {"", "max"}, {"test-model", ""}} {
			model, effort := settings.model, settings.effort
			t.Run(provider.name+"/"+model+"/"+effort, func(t *testing.T) {
				dir := t.TempDir()
				script := "#!/bin/sh\ncat >/dev/null\nprintf '%s\\n' \"$@\" > arguments\nprintf '%s\\n' '" + provider.events + "'\n"
				if err := os.WriteFile(filepath.Join(dir, provider.name), []byte(script), 0700); err != nil {
					t.Fatal(err)
				}
				t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
				runner := provider.newRunner(model, effort)
				for turn := 0; turn < 2; turn++ {
					if _, err := runner.Run(dir, "prompt", nil); err != nil {
						t.Fatal(err)
					}
					data, err := os.ReadFile(filepath.Join(dir, "arguments"))
					if err != nil {
						t.Fatal(err)
					}
					arguments := "\n" + string(data)
					if model == "" && strings.Contains(arguments, "\n--model\n") {
						t.Fatalf("default overridden: %s", data)
					}
					if model != "" && !strings.Contains(arguments, "\n--model\ntest-model\n") {
						t.Fatalf("model missing on turn %d: %s", turn, data)
					}
					level := strings.ToLower(strings.TrimSpace(effort))
					flag, value := "--effort", level
					if provider.name == "codex" {
						flag, value = "--config", "model_reasoning_effort="+strconv.Quote(level)
					}
					if level == "" && strings.Contains(arguments, "\n"+flag+"\n") {
						t.Fatalf("default effort overridden: %s", data)
					}
					if level != "" && !strings.Contains(arguments, "\n"+flag+"\n"+value+"\n") {
						t.Fatalf("effort missing on turn %d: %s", turn, data)
					}
				}
			})
		}
	}
}
