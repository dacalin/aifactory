package commands

import (
	"github.com/dacalin/aifactory/internal/agents"
	"github.com/spf13/cobra"
)

func NewRootCommand(newAgent func() (agents.Agent, error)) *cobra.Command {
	root := &cobra.Command{
		Use:           "aifactory",
		Version:       Version,
		Short:         "Manage project agents and rules",
		Args:          cobra.NoArgs,
		RunE:          func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.CompletionOptions.DisableDefaultCmd = true
	root.AddCommand(newInitCommand(), newRuleCommand(newAgent))
	return root
}
