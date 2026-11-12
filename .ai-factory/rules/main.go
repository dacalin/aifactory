package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func main() {
	if err := rootCommand().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func rootCommand() *cobra.Command {
	var root, dataDir, rulesDir string
	var list bool
	command := &cobra.Command{
		Use:           "checkrules [all|check-name]",
		Short:         "Run project rules",
		Args:          cobra.MaximumNArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(command *cobra.Command, args []string) error {
			names := checkNames()
			if list {
				for _, name := range names {
					fmt.Fprintln(command.OutOrStdout(), name)
				}
				return nil
			}
			if len(args) == 1 && args[0] != "all" {
				if _, exists := checks[args[0]]; !exists {
					return fmt.Errorf("unknown check %q; use --list", args[0])
				}
				names = args
			}
			projectRoot, err := findRoot(root)
			if err != nil {
				return err
			}
			dataDir, rulesDir, err = checkerDirectories(projectRoot, dataDir, rulesDir)
			if err != nil {
				return err
			}
			repo, err := newRepository(projectRoot, dataDir, rulesDir)
			if err != nil {
				return err
			}
			return runChecks(repo, names, command.OutOrStdout())
		},
	}
	command.Flags().StringVar(&root, "root", "", "Project directory; defaults to the nearest factory.yaml or go.mod")
	command.Flags().StringVar(&dataDir, "data-dir", "", "Project-relative generated artifact directory to exclude")
	command.Flags().StringVar(&rulesDir, "rules-dir", "", "Project-relative rule documents")
	command.Flags().BoolVar(&list, "list", false, "List available checks")
	return command
}

func findRoot(requested string) (string, error) {
	if requested != "" {
		return filepath.Abs(requested)
	}
	current, err := os.Getwd()
	if err != nil {
		return "", err
	}
	fallback := ""
	for {
		if _, err := os.Stat(filepath.Join(current, "factory.yaml")); err == nil {
			return current, nil
		}
		if _, err := os.Stat(filepath.Join(current, "go.mod")); err == nil && fallback == "" {
			fallback = current
		}
		parent := filepath.Dir(current)
		if parent == current {
			if fallback != "" {
				return fallback, nil
			}
			return "", fmt.Errorf("project root not found; pass --root")
		}
		current = parent
	}
}
