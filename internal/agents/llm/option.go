package llm

import "github.com/dacalin/aifactory/internal/agents/llm/runnercli"

// Option configures a provider runner.
type Option = runnercli.Option

// WithWritableDirs restricts writes to the supplied project subdirectories.
// At least one directory is required. The first also holds provider state.
func WithWritableDirs(dirs ...string) Option {
	return runnercli.WithWritableDirs(dirs...)
}
