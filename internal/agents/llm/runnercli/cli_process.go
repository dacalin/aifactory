package runnercli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

func conversationDirectory(projectDir, current string) (string, error) {
	dir, err := filepath.Abs(projectDir)
	if err != nil {
		return "", fmt.Errorf("resolve project directory: %w", err)
	}
	if current != "" && current != dir {
		return "", fmt.Errorf("conversation already belongs to %s", current)
	}
	return dir, nil
}

func RunCLI(ctx context.Context, cmd *exec.Cmd, readReply func(io.Reader) (string, error)) (string, error) {
	// Provider tools inherit this group so cancellation also reaches them on macOS and Linux.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return killProcessGroup(cmd.Process.Pid) }
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("start %s CLI: %w", cmd.Args[0], err)
	}
	stop := context.AfterFunc(ctx, func() { _ = stdout.Close() })
	defer stop()
	reply, readErr := readReply(stdout)
	if readErr != nil {
		_ = cmd.Cancel()
	}
	waitErr := cmd.Wait()
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if readErr != nil {
		return "", readErr
	}
	if waitErr != nil {
		return "", fmt.Errorf("run %s CLI: %w", cmd.Args[0], waitErr)
	}
	return reply, nil
}

func killProcessGroup(pid int) error {
	err := syscall.Kill(-pid, syscall.SIGKILL)
	if errors.Is(err, syscall.ESRCH) {
		return os.ErrProcessDone
	}
	return err
}
