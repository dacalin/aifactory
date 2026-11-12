package agents

import _ "embed"

// DefaultRulePrompt is written by init only when the project prompt is missing.
//
//go:embed prompts/rule-agent.md
var DefaultRulePrompt string
