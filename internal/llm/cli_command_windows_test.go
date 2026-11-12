package llm

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestWindowsNPMShimUsesNodeWithoutShell(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("Node is required for npm launcher test")
	}
	dir := t.TempDir()
	for provider, entry := range map[string]string{
		"codex":  "@openai/codex/bin/codex.js",
		"claude": "@anthropic-ai/claude-code/cli.js",
	} {
		shim := filepath.Join(dir, provider+".cmd")
		if err := os.WriteFile(shim, []byte("exit /b 99\n"), 0600); err != nil {
			t.Fatal(err)
		}
		name := filepath.Join(dir, "node_modules", filepath.FromSlash(entry))
		if err := os.MkdirAll(filepath.Dir(name), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(name, []byte("console.log(JSON.stringify(process.argv.slice(2)))"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	for _, provider := range []string{"codex", "claude"} {
		cmd, err := providerCommand(provider, "--model", "literal & value")
		if err != nil {
			t.Fatal(err)
		}
		output, err := cmd.CombinedOutput()
		if err != nil || strings.TrimSpace(string(output)) != `["--model","literal & value"]` {
			t.Fatalf("launcher: %q, %v", output, err)
		}
	}
}
