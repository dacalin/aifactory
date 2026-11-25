package commands

import (
	"context"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/dacalin/aifactory/internal/agents"
	"github.com/spf13/cobra"
)

func NewRootCommand(newAgent func() (agents.Agent, error), runServer func(context.Context, io.Writer, bool) error) *cobra.Command {
	root := &cobra.Command{
		Use:     "aifactory",
		Version: Version,
		Short:   "Manage projects, create rules, and run checks",
		Long:    "AI Factory provides a local project manager and turns project requirements into documented rules and executable checks.\n\nStart the project manager with serve. Initialize project rule configuration with init, create or update a rule with rule <prompt>,\nand execute the current project or worktree checks with rule review.",
		Example: `  aifactory init
  aifactory serve
  aifactory rule "Keep interfaces in separate files"
  aifactory rule review`,
		Args:          cobra.NoArgs,
		RunE:          func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.CompletionOptions.DisableDefaultCmd = true
	serve := &cobra.Command{
		Use:   "serve",
		Short: "Open the local AI Factory project manager",
		Long:  "Start the local project manager, register project folders, and keep the server running until interrupted.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer stop()
			noBrowser, _ := cmd.Flags().GetBool("no-browser")
			return runServer(ctx, cmd.OutOrStdout(), noBrowser)
		},
	}
	serve.Flags().Bool("no-browser", false, "Do not open the browser automatically")
	root.AddCommand(newInitCommand(), newRuleCommand(newAgent), serve)
	return root
}
