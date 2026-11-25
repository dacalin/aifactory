package codex

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/dacalin/aifactory/internal/agents"
	"github.com/dacalin/aifactory/internal/agents/llm/runnercli"
)

type CodexCLIRunner struct {
	model     string
	effort    string
	sessionID string
	cli       *runnercli.CLI
}

var _ agents.AgentRunner = (*CodexCLIRunner)(nil)

func NewRunner(model, effort string, options ...runnercli.Option) *CodexCLIRunner {
	return &CodexCLIRunner{
		model: strings.TrimSpace(model), effort: strings.ToLower(strings.TrimSpace(effort)),
		cli: runnercli.NewCLI("codex", "CODEX_HOME", []string{"config.toml", "auth.json"}, options...),
	}
}

func (runner *CodexCLIRunner) Close() error {
	return runner.cli.Close()
}

func (runner *CodexCLIRunner) Run(ctx context.Context, projectDir, prompt string, onMessage func(string)) (string, error) {
	mode := "workspace-write"
	if runner.cli.HasWriteRestriction() {
		// The enclosing OS sandbox already restricts the CLI and all its children.
		mode = "danger-full-access"
	}
	args := []string{"--ask-for-approval", "never", "--sandbox", mode}
	if runner.model != "" {
		args = append(args, "--model", runner.model)
	}
	if runner.effort != "" {
		args = append(args, "--config", "model_reasoning_effort="+strconv.Quote(runner.effort))
	}
	args = append(args, "exec")
	if runner.sessionID != "" {
		args = append(args, "resume", "--json", "--skip-git-repo-check", runner.sessionID, "-")
	} else {
		args = append(args, "--json", "--skip-git-repo-check", "-")
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
		return "", fmt.Errorf("Codex returned no conversation ID or assistant reply")
	}
	return reply, nil
}

func (runner *CodexCLIRunner) readReply(stdout io.Reader, onMessage func(string)) (string, error) {
	if onMessage == nil {
		onMessage = func(string) {}
	}
	decoder := json.NewDecoder(stdout)
	var reply strings.Builder
	for {
		var event struct {
			Type     string `json:"type"`
			ThreadID string `json:"thread_id"`
			Message  string `json:"message"`
			Item     struct{ Type, Text string }
			Error    struct{ Message string }
		}
		if err := decoder.Decode(&event); err == io.EOF {
			return reply.String(), nil
		} else if err != nil {
			return "", fmt.Errorf("read Codex events: %w", err)
		}
		switch event.Type {
		case "turn.started":
			onMessage("Agent is working...")
		case "item.started":
			if event.Item.Type == "command_execution" {
				onMessage("Running a project command...")
			}
		case "thread.started":
			runner.sessionID = event.ThreadID
		case "item.completed":
			if event.Item.Type == "agent_message" {
				reply.WriteString(event.Item.Text + "\n")
				onMessage(event.Item.Text)
			}
		case "turn.failed":
			return "", fmt.Errorf("Codex turn failed: %s", event.Error.Message)
		case "error":
			return "", fmt.Errorf("Codex error: %s", event.Message)
		}
	}
}
