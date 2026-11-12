package agents

// AgentRunner maintains a conversation for one project.
// Run streams messages and progress through onMessage and returns the full reply.
// Earlier turns remain in context. A nil callback disables streaming.
type AgentRunner interface {
	Run(projectDir, prompt string, onMessage func(string)) (string, error)
}
