package runnercli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/dacalin/aifactory/internal/sandbox"
)

// CLI handles working directory, writable paths, and provider session storage.
type CLI struct {
	program      string
	homeVariable string
	inputFiles   []string
	writableDirs []string
	projectDir   string
	homeDir      string
	tempDir      string
	cleanup      func() error
	closed       bool
}

func NewCLI(program, homeVariable string, inputFiles []string, options ...Option) *CLI {
	cli := &CLI{program: program, homeVariable: homeVariable, inputFiles: inputFiles}
	for _, option := range options {
		option(cli)
	}
	return cli
}

func (cli *CLI) HasWriteRestriction() bool {
	return cli.writableDirs != nil
}

func (cli *CLI) Close() error {
	cli.closed = true
	if cli.cleanup == nil {
		return nil
	}
	cleanup := cli.cleanup
	cli.cleanup = nil
	return cleanup()
}

func (cli *CLI) Command(ctx context.Context, project string, args ...string) (*exec.Cmd, error) {
	if cli.closed {
		return nil, os.ErrClosed
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	project, err := conversationDirectory(project, cli.projectDir)
	if err != nil {
		return nil, err
	}
	cli.projectDir = project
	if !cli.HasWriteRestriction() {
		cmd := exec.CommandContext(ctx, cli.program, args...)
		cmd.Dir = project
		return cmd, nil
	}

	boundaries, err := cli.boundaries(project)
	if err != nil {
		return nil, err
	}
	cmd, err := sandbox.CommandWithWritableDirs(ctx, boundaries, project, cli.program, args...)
	if err != nil {
		return nil, err
	}
	if cli.homeDir == "" {
		cli.homeDir, cli.tempDir, cli.cleanup, err = providerState(boundaries[0], cli.program, cli.homeVariable, cli.inputFiles)
		if err != nil {
			return nil, err
		}
	}
	cmd.Env = append(cmd.Environ(), cli.homeVariable+"="+cli.homeDir,
		"TMPDIR="+cli.tempDir, "GOTMPDIR="+cli.tempDir, "GOCACHE="+filepath.Join(cli.tempDir, "go-build"))
	return cmd, nil
}

func (cli *CLI) boundaries(project string) ([]string, error) {
	if len(cli.writableDirs) == 0 {
		return nil, fmt.Errorf("WithWritableDirs requires at least one project subdirectory")
	}
	boundaries := make([]string, 0, len(cli.writableDirs))
	for _, path := range cli.writableDirs {
		boundary, err := sandbox.Directory(project, path)
		if err != nil {
			return nil, err
		}
		boundaries = append(boundaries, boundary)
	}
	return boundaries, nil
}
