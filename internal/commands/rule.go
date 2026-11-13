package commands

import (
	"fmt"
	"strings"

	"github.com/dacalin/aifactory/internal/agents"
	"github.com/spf13/cobra"
)

func newRuleCommand(newAgent func() (agents.Agent, error)) *cobra.Command {
	command := &cobra.Command{
		Use:   "rule [prompt]",
		Short: "Create, update, or review project rules",
		Long:  "Provide a rule description to create or update a rule through a conversation\nwith the agent configured in factory.yaml. Run this from the project root.\nThe agent reads its editable prompt from <dir>/prompts/rule-agent.md.\n\nProgress and replies are streamed. Send follow-up messages to continue;\ntype /exit or end input to finish. Use the review subcommand to run checks.",
		Example: `  aifactory rule "Keep interfaces in separate files"
  aifactory rule "Limit functions to 60 lines"
  aifactory rule review
  aifactory rule review interface-files`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			prompt := strings.TrimSpace(strings.Join(args, " "))
			if prompt == "" {
				return fmt.Errorf("provide a rule description")
			}
			agent, err := newAgent()
			if err != nil {
				return err
			}
			return runConversation(agent, prompt, cmd.InOrStdin(), cmd.OutOrStdout())
		},
	}
	command.AddCommand(newReviewCommand())
	return command
}
