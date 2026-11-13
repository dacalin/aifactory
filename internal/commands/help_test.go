package commands

import (
	"bytes"
	"strings"
	"testing"

	"github.com/dacalin/aifactory/internal/agents"
)

func TestCommandHelpExplainsUsageAndSubcommands(t *testing.T) {
	for _, test := range []struct {
		args      []string
		fragments []string
	}{
		{[]string{"--help"}, []string{"aifactory init", "aifactory rule review", "Create, update, or review project rules"}},
		{[]string{"rule", "--help"}, []string{`aifactory rule "Keep interfaces in separate files"`, "aifactory rule review interface-files", "review", "/exit"}},
		{[]string{"rule", "review", "--help"}, []string{"review [check-id|all]", "Run all checks by default", "aifactory rule review --list", "nonzero exit status"}},
		{[]string{"init", "--help"}, []string{"Existing configuration and edited prompts are preserved", "<dir>/prompts/rule-agent.md", "aifactory init"}},
	} {
		t.Run(strings.Join(test.args, " "), func(t *testing.T) {
			command := NewRootCommand(func() (agents.Agent, error) { t.Fatal("help constructed an agent"); return nil, nil })
			var output bytes.Buffer
			command.SetOut(&output)
			command.SetArgs(test.args)
			if err := command.Execute(); err != nil {
				t.Fatal(err)
			}
			for _, fragment := range test.fragments {
				if !strings.Contains(output.String(), fragment) {
					t.Fatalf("help %v is missing %q:\n%s", test.args, fragment, output.String())
				}
			}
		})
	}
}
