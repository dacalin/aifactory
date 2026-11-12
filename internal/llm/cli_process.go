package llm

import (
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
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

func runCLI(cmd *exec.Cmd, readReply func(io.Reader) (string, error)) (string, error) {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("start %s CLI: %w", cmd.Args[0], err)
	}
	reply, readErr := readReply(stdout)
	if readErr != nil {
		_ = cmd.Process.Kill()
	}
	waitErr := cmd.Wait()
	if readErr != nil {
		return "", readErr
	}
	if waitErr != nil {
		return "", fmt.Errorf("run %s CLI: %w", cmd.Args[0], waitErr)
	}
	return reply, nil
}
