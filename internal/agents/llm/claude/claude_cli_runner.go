package claude

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/dacalin/aifactory/internal/agents"
	"github.com/dacalin/aifactory/internal/agents/llm/runnercli"
)

type ClaudeCLIRunner struct {
	model     string
	effort    string
	sessionID string
	cli       *runnercli.CLI
}

var _ agents.AgentRunner = (*ClaudeCLIRunner)(nil)

func NewRunner(model, effort string, options ...runnercli.Option) *ClaudeCLIRunner {
	return &ClaudeCLIRunner{
		model: strings.TrimSpace(model), effort: strings.ToLower(strings.TrimSpace(effort)),
		cli: runnercli.NewCLI("claude", "CLAUDE_CONFIG_DIR", []string{"settings.json", ".credentials.json", ".claude.json"}, options...),
	}
}

func (runner *ClaudeCLIRunner) Close() error {
	return runner.cli.Close()
}

func (runner *ClaudeCLIRunner) Run(ctx context.Context, projectDir, prompt string, onMessage func(string)) (string, error) {
	args := []string{
		"--print", "--verbose", "--output-format", "stream-json",
		"--permission-mode", "acceptEdits",
		"--allowedTools", "Read,Glob,Grep,Edit,Write,Bash",
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
	cmd, err := runner.cli.Command(ctx, projectDir, args...)
	if err != nil {
		return "", err
	}
	cmd.Stdin = strings.NewReader(prompt)
	cmd.Stderr = os.Stderr
	reply, err := runnercli.RunCLI(ctx, cmd, func(stdout io.Reader) (string, error) {
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
