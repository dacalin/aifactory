package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newReviewCommand() *cobra.Command {
	var list bool
	command := &cobra.Command{
		Use:   "review [check-id|all]",
		Short: "Run rule checks in the current project or worktree",
		Long:  "Run all checks by default, or provide one check ID. Review executes the\ncommand declared in checks.run and reports its output and failures.\nThe optional checks.build command runs first; both use the rule directory.\n\nRun from the project root or a subdirectory. Review finds factory.yaml within\nthe current project/worktree and includes uncommitted changes in its checks.\nA failed build or check returns a nonzero exit status.",
		Example: `  aifactory rule review
  aifactory rule review all
  aifactory rule review interface-files
  aifactory rule review --list`,
		Args: cobra.MaximumNArgs(1),
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
	command.Flags().BoolVar(&list, "list", false, "List available check IDs instead of running checks")
	return command
}
