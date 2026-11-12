package commands

import (
	"fmt"
	"strings"

	"github.com/dacalin/aifactory/internal/agents"
	"github.com/spf13/cobra"
)

func newRuleCommand(newAgent func() (agents.Agent, error)) *cobra.Command {
	command := &cobra.Command{
		Use:   "rule <prompt>",
		Short: "Create or update a project rule through a conversation",
		Args:  cobra.MinimumNArgs(1),
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
