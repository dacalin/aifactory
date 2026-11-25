package commands

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dacalin/aifactory/internal/config"
	"github.com/dacalin/aifactory/internal/sandbox"

	"github.com/spf13/cobra"
)

func runReview(command *cobra.Command, args []string) error {
	root, err := reviewProjectDir()
	if err != nil {
		return err
	}
	cfg, err := config.Load(root)
	if err != nil {
		return err
	}
	commands := cfg.Checks
	if len(commands.Run) == 0 {
		return fmt.Errorf("checks.run is not configured in factory.yaml; ask RuleAgent to configure the shared checker")
	}
	rules, err := sandbox.Directory(cfg.ProjectDir, cfg.RulesDir())
	if err != nil {
		return err
	}
	workspace, err := sandbox.RuntimeDirectory(rules)
	if err != nil {
		return err
	}
	rulesRoot, err := os.OpenRoot(rules)
	if err != nil {
		return err
	}
	defer rulesRoot.Close()
	runtimeRoot, err := rulesRoot.OpenRoot(filepath.Base(workspace))
	if err != nil {
		return err
	}
	defer runtimeRoot.Close()
	temporary, err := os.MkdirTemp(workspace, "review-")
	if err != nil {
		return err
	}
	defer runtimeRoot.RemoveAll(filepath.Base(temporary))
	if len(commands.Build) > 0 {
		if err := executeChecker(command, root, rules, temporary, commands.Build, nil); err != nil {
			return fmt.Errorf("build configured checker: %w", err)
		}
	}
	if err := executeChecker(command, root, rules, temporary, commands.Run, args); err != nil {
		return fmt.Errorf("rule review failed: %w", err)
	}
	return nil
}

func executeChecker(command *cobra.Command, root, rules, temporary string, argv, args []string) error {
	program := argv[0]
	if !filepath.IsAbs(program) && strings.ContainsRune(program, '/') {
		program = filepath.Join(rules, program)
	}
	arguments := append(append([]string{}, argv[1:]...), args...)
	process, err := sandbox.Command(command.Context(), rules, rules, program, arguments...)
	if err != nil {
		return err
	}
	process.Env = append(process.Environ(), "TMPDIR="+temporary, "GOTMPDIR="+temporary,
		"GOCACHE="+filepath.Join(filepath.Dir(temporary), "go-build"), "AIFACTORY_PROJECT_DIR="+root)
	process.Stdout = command.OutOrStdout()
	process.Stderr = command.ErrOrStderr()
	return process.Run()
}

func reviewProjectDir() (string, error) {
	directory, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(directory, "factory.yaml")); err == nil {
			return directory, nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		// A worktree has a .git file; a regular checkout has a .git directory.
		if _, err := os.Stat(filepath.Join(directory, ".git")); err == nil {
			break
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			break
		}
		directory = parent
	}
	return "", fmt.Errorf("factory.yaml not found in this project/worktree; run aifactory init at its root")
}
