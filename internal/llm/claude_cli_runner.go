package llm

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/dacalin/aifactory/internal/agents"
)

type ClaudeCLIRunner struct {
	model      string
	effort     string
	sessionID  string
	projectDir string
}

var _ agents.AgentRunner = (*ClaudeCLIRunner)(nil)

func NewClaudeCLIRunner(model, effort string) *ClaudeCLIRunner {
	return &ClaudeCLIRunner{model: strings.TrimSpace(model), effort: strings.ToLower(strings.TrimSpace(effort))}
}

func (runner *ClaudeCLIRunner) Run(projectDir, prompt string, onMessage func(string)) (string, error) {
	dir, err := conversationDirectory(projectDir, runner.projectDir)
	if err != nil {
		return "", err
	}
	runner.projectDir = dir
	args := []string{
		"--print", "--verbose", "--output-format", "stream-json",
		"--permission-mode", "acceptEdits",
		"--tools", "Read,Glob,Grep,Edit,Write,Bash",
		"--allowedTools", "Read,Glob,Grep,Edit,Write,Bash",
		"--strict-mcp-config", "--mcp-config", "{\"mcpServers\":{}}",
	}
	if runner.model != "" {
		args = append(args, "--model", runner.model)
	}
	if runner.effort != "" {
		args = append(args, "--effort", runner.effort)
	}
	if runner.sessionID != "" {
		args = append(args, "--resume", runner.sessionID)
	}
	cmd, err := providerCommand("claude", args...)
	if err != nil {
		return "", err
	}
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader(prompt)
	cmd.Stderr = os.Stderr
	reply, err := runCLI(cmd, func(stdout io.Reader) (string, error) {
		return runner.readReply(stdout, onMessage)
	})
	if err != nil {
		return "", err
	}
	if runner.sessionID == "" || strings.TrimSpace(reply) == "" {
		return "", fmt.Errorf("Claude Code returned no conversation ID or assistant reply")
	}
	return reply, nil
}

func (runner *ClaudeCLIRunner) readReply(stdout io.Reader, onMessage func(string)) (string, error) {
	if onMessage == nil {
		onMessage = func(string) {}
	}
	decoder := json.NewDecoder(stdout)
	var turn claudeTurn
	for {
		var event claudeEvent
		err := decoder.Decode(&event)
		if err == io.EOF {
			return turn.finish()
		}
		if err != nil {
			return "", fmt.Errorf("read Claude Code events: %w", err)
		}
		if event.ParentToolUseID != "" {
			continue
		}
		if event.SessionID != "" {
			runner.sessionID = event.SessionID
		}
		if err := turn.consume(event, onMessage); err != nil {
			return "", err
		}
	}
}
