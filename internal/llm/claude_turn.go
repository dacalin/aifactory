package llm

import (
	"fmt"
	"strings"
)

type claudeTurn struct {
	reply       strings.Builder
	lastMessage string
	completed   bool
}

func (turn *claudeTurn) consume(event claudeEvent, onMessage func(string)) error {
	switch event.Type {
	case "system":
		if event.Subtype == "init" {
			onMessage("Agent is working...")
		}
	case "assistant":
		turn.assistant(event, onMessage)
	case "result":
		return turn.result(event, onMessage)
	}
	return nil
}

func (turn *claudeTurn) assistant(event claudeEvent, onMessage func(string)) {
	var text strings.Builder
	for _, block := range event.Message.Content {
		switch block.Type {
		case "text":
			text.WriteString(block.Text + "\n")
			onMessage(block.Text)
		case "tool_use":
			onMessage("Working on project files...")
		}
	}
	if text.Len() > 0 {
		turn.lastMessage = strings.TrimSpace(text.String())
		turn.reply.WriteString(text.String())
	}
}

func (turn *claudeTurn) result(event claudeEvent, onMessage func(string)) error {
	turn.completed = true
	if event.IsError {
		message := event.Result
		if len(event.Errors) > 0 {
			message = strings.Join(event.Errors, "\n")
		}
		return fmt.Errorf("Claude Code turn failed (%s): %s", event.Subtype, message)
	}
	text := strings.TrimSpace(event.Result)
	if text != "" && text != turn.lastMessage {
		onMessage(event.Result)
		turn.reply.WriteString(event.Result + "\n")
	}
	return nil
}

func (turn *claudeTurn) finish() (string, error) {
	if !turn.completed {
		return "", fmt.Errorf("Claude Code stream ended without a result")
	}
	return turn.reply.String(), nil
}
