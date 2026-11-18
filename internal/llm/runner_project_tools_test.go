package llm

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dacalin/aifactory/internal/agents"
)

func TestRunnersPreserveProjectToolConfiguration(t *testing.T) {
	for _, provider := range []struct {
		name      string
		newRunner func() agents.AgentRunner
		config    string
		content   string
		events    string
	}{
		{"claude", func() agents.AgentRunner { return NewClaudeCLIRunner("", "") }, ".mcp.json",
			`{"mcpServers":{"project-tools":{"command":"project-mcp-server"}}}`,
			`{"type":"system","session_id":"tools-test"}
{"type":"result","subtype":"success","result":"done","is_error":false}`},
		{"codex", func() agents.AgentRunner { return NewCodexCLIRunner("", "") }, ".codex/config.toml",
			"[mcp_servers.project-tools]\ncommand = \"project-mcp-server\"\n",
			`{"type":"thread.started","thread_id":"tools-test"}
{"type":"item.completed","item":{"type":"agent_message","text":"done"}}
{"type":"turn.completed"}`},
	} {
		t.Run(provider.name, func(t *testing.T) {
			dir := t.TempDir()
			configPath := filepath.Join(dir, provider.config)
			if err := os.MkdirAll(filepath.Dir(configPath), 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(configPath, []byte(provider.content), 0600); err != nil {
				t.Fatal(err)
			}
			script := "#!/bin/sh\nset -eu\ncat >/dev/null\ntest -f '" + provider.config + "'\nprintf '%s\\n' \"$@\" > arguments\nprintf '%s\\n' '" + provider.events + "'\n"
			if err := os.WriteFile(filepath.Join(dir, provider.name), []byte(script), 0700); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
			runner := provider.newRunner()
			for turn := 0; turn < 2; turn++ {
				if _, err := runner.Run(dir, "use project tools", nil); err != nil {
					t.Fatal(err)
				}
				data, err := os.ReadFile(filepath.Join(dir, "arguments"))
				if err != nil {
					t.Fatal(err)
				}
				for _, forbidden := range []string{"--strict-mcp-config", "--mcp-config", "--tools", "--disable-slash-commands", "--bare", "--safe-mode", "--setting-sources", "mcp_servers", "--dangerously-skip-permissions"} {
					if strings.Contains(string(data), forbidden) {
						t.Fatalf("turn %d overrides project tools or permissions with %q: %s", turn, forbidden, data)
					}
				}
				current, err := os.ReadFile(configPath)
				if err != nil || string(current) != provider.content {
					t.Fatalf("project configuration changed: %q, %v", current, err)
				}
			}
		})
	}
}
