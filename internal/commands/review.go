package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newReviewCommand() *cobra.Command {
	var list bool
	command := &cobra.Command{
		Use:   "review [check-id|all]",
		Short: "Run the current worktree's rule checks without starting an agent",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if list && len(args) > 0 {
				return fmt.Errorf("use --list without a check ID")
			}
			if list {
				args = []string{"--list"}
			} else if len(args) == 0 {
				args = []string{"all"}
			}
			return runReview(cmd, args)
		},
	}
	command.Flags().BoolVar(&list, "list", false, "List available checks")
	return command
}
