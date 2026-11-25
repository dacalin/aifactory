package agents

import "context"

// Agent handles conversation turns and streams messages to the caller.
type Agent interface {
	// Converse sends a turn. A nil onMessage callback disables streaming.
	Converse(prompt string, onMessage func(string)) (string, error)
	// ConverseContext sends a turn that the caller can cancel.
	ConverseContext(ctx context.Context, prompt string, onMessage func(string)) (string, error)
}
