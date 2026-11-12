package commands

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

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
	rules := filepath.Join(root, cfg.RulesDir())
	runner := filepath.Join(rules, reviewExecutable())
	// Rebuild Go checks from this checkout; ignored binaries may be stale or absent.
	if _, err := os.Stat(filepath.Join(rules, "go.mod")); err == nil {
		if err := buildReviewRunner(command, cfg, rules, runner); err != nil {
			return err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if _, err := os.Stat(runner); err != nil {
		return fmt.Errorf("rule runner unavailable at %s; create a rule or build its shared checkrules entry point: %w", runner, err)
	}
	process := exec.CommandContext(command.Context(), runner, args...)
	process.Dir = root
	process.Stdout = command.OutOrStdout()
	process.Stderr = command.ErrOrStderr()
	if err := process.Run(); err != nil {
		return fmt.Errorf("rule review failed: %w", err)
	}
	return nil
}

func buildReviewRunner(command *cobra.Command, cfg config.Factory, rules, runner string) error {
	workspace := filepath.Join(cfg.ProjectDir, cfg.TemporaryDir())
	if err := os.MkdirAll(workspace, 0755); err != nil {
		return err
	}
	temporary, err := os.MkdirTemp(workspace, "review-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(temporary)
	process := exec.CommandContext(command.Context(), "go", "build", "-o", runner, ".")
	process.Dir = rules
	process.Env = append(os.Environ(), "TMPDIR="+temporary)
	process.Stdout = command.OutOrStdout()
	process.Stderr = command.ErrOrStderr()
	if err := process.Run(); err != nil {
		return fmt.Errorf("build current worktree's rule runner: %w", err)
	}
	return nil
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

func reviewExecutable() string {
	if runtime.GOOS == "windows" {
		return "checkrules.exe"
	}
	return "checkrules"
}
