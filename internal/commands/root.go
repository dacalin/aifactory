package commands

import (
	"github.com/dacalin/aifactory/internal/agents"
	"github.com/spf13/cobra"
)

func NewRootCommand(newAgent func() (agents.Agent, error)) *cobra.Command {
	root := &cobra.Command{
		Use:     "aifactory",
		Version: Version,
		Short:   "Create project rules and run their checks",
		Long:    "AI Factory turns project requirements into documented rules and executable checks.\n\nInitialize a project with init, create or update a rule with rule <prompt>,\nand execute the current project or worktree checks with rule review.",
		Example: `  aifactory init
  aifactory rule "Keep interfaces in separate files"
  aifactory rule review`,
		Args:          cobra.NoArgs,
		RunE:          func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.CompletionOptions.DisableDefaultCmd = true
	root.AddCommand(newInitCommand(), newRuleCommand(newAgent))
	return root
}
