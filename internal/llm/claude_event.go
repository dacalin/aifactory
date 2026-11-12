package llm

type claudeEvent struct {
	Type            string
	Subtype         string
	SessionID       string `json:"session_id"`
	ParentToolUseID string `json:"parent_tool_use_id"`
	IsError         bool   `json:"is_error"`
	Result          string
	Errors          []string
	Message         struct {
		Content []struct{ Type, Text, Name string }
	}
}
