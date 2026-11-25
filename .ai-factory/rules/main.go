package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	if managedRoot := strings.TrimSpace(os.Getenv("AIFACTORY_PROJECT_DIR")); managedRoot != "" {
		return filepath.Abs(managedRoot)
	}
	current, err := os.Getwd()
	if err != nil {
		return "", err
	}
	if root, found, err := ancestorWith(current, "factory.yaml"); err != nil {
		return "", err
	} else if found {
		return root, nil
	}
	if executable, err := os.Executable(); err == nil {
		if resolved, resolveErr := filepath.EvalSymlinks(executable); resolveErr == nil {
			executable = resolved
		}
		if root, found, findErr := ancestorWith(filepath.Dir(executable), "factory.yaml"); findErr != nil {
			return "", findErr
		} else if found {
			return root, nil
		}
	}
	if root, found, err := ancestorWith(current, "go.mod"); err != nil {
		return "", err
	} else if found {
		return root, nil
	}
	return "", fmt.Errorf("project root not found; pass --root")
}

func ancestorWith(start, name string) (string, bool, error) {
	current, err := filepath.Abs(start)
	if err != nil {
		return "", false, err
	}
	for {
		if _, err := os.Stat(filepath.Join(current, name)); err == nil {
			return current, true, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", false, nil
		}
		current = parent
	}
}
