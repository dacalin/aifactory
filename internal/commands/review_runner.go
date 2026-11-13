package commands

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/dacalin/aifactory/internal/config"

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
	workspace := filepath.Join(root, cfg.TemporaryDir())
	if err := os.MkdirAll(workspace, 0755); err != nil {
		return err
	}
	temporary, err := os.MkdirTemp(workspace, "review-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(temporary)
	if len(commands.Build) > 0 {
		if err := executeChecker(command, cfg, commands.Build, nil, temporary); err != nil {
			return fmt.Errorf("build configured checker: %w", err)
		}
	}
	if err := executeChecker(command, cfg, commands.Run, args, temporary); err != nil {
		return fmt.Errorf("rule review failed: %w", err)
	}
	return nil
}

func executeChecker(command *cobra.Command, cfg config.Factory, argv, args []string, temporary string) error {
	rules := filepath.Join(cfg.ProjectDir, cfg.RulesDir())
	program := argv[0]
	if !filepath.IsAbs(program) && strings.ContainsRune(program, '/') {
		program = filepath.Join(rules, program)
	}
	arguments := append(append([]string{}, argv[1:]...), args...)
	process := exec.CommandContext(command.Context(), program, arguments...)
	process.Dir = rules
	process.Env = append(os.Environ(), "TMPDIR="+temporary, "AIFACTORY_PROJECT_DIR="+cfg.ProjectDir)
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
