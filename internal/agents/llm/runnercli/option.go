package runnercli

// Option configures the CLI shared by a provider runner.
type Option func(*CLI)

// WithWritableDirs restricts writes to the named subdirectories of the project.
// The first directory also stores the provider's private runtime state.
// At least one directory is required when using this option.
func WithWritableDirs(dirs ...string) Option {
	dirs = append([]string{}, dirs...)
	return func(cli *CLI) {
		cli.writableDirs = dirs
	}
}
