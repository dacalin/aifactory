package llm

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// npm installs .cmd shims on Windows; invoke their JS entry point without a shell.
func providerCommand(provider string, args ...string) (*exec.Cmd, error) {
	program, err := exec.LookPath(provider)
	if err != nil {
		return nil, fmt.Errorf("find %s CLI on PATH: %w", provider, err)
	}
	extension := strings.ToLower(filepath.Ext(program))
	if runtime.GOOS != "windows" || (extension != ".cmd" && extension != ".bat") {
		return exec.Command(program, args...), nil
	}
	var entry string
	switch provider {
	case "codex":
		entry = "@openai/codex/bin/codex.js"
	case "claude":
		entry = "@anthropic-ai/claude-code/cli.js"
	default:
		return nil, fmt.Errorf("unsupported Windows CLI shim: %s", provider)
	}
	entry = filepath.Join(filepath.Dir(program), "node_modules", filepath.FromSlash(entry))
	if _, err := os.Stat(entry); err != nil {
		return nil, fmt.Errorf("find %s npm entry point (or install the native CLI): %w", provider, err)
	}
	return exec.Command("node", append([]string{entry}, args...)...), nil
}
