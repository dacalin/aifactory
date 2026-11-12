package commands

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/dacalin/aifactory/internal/agents"
	"github.com/dacalin/aifactory/internal/config"

	"github.com/spf13/cobra"
)

func newInitCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Create missing factory.yaml and default prompts",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return initProject(cmd.OutOrStdout())
		},
	}
}

func initProject(output io.Writer) error {
	if err := createInitialFile("factory.yaml", config.DefaultYAML, output); err != nil {
		return err
	}
	cfg, err := config.Load(".")
	if err != nil {
		return err
	}
	return createInitialFile(cfg.RulePromptPath(), agents.DefaultRulePrompt, output)
}

// Exclusive creation preserves existing configuration and edited prompts.
func createInitialFile(name, content string, output io.Writer) error {
	if err := os.MkdirAll(filepath.Dir(name), 0755); err != nil {
		return err
	}
	file, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if errors.Is(err, os.ErrExist) {
		fmt.Fprintf(output, "%s already exists; preserved\n", name)
		return nil
	}
	if err != nil {
		return err
	}
	_, writeErr := file.WriteString(content)
	if err := errors.Join(writeErr, file.Close()); err != nil {
		return errors.Join(err, os.Remove(name))
	}
	fmt.Fprintf(output, "Created %s\n", name)
	return nil
}
