package git

import (
	"context"
	"os/exec"
	"slices"
	"strings"
)

func outputBytes(ctx context.Context, directory string, args ...string) ([]byte, error) {
	return command(ctx, directory, args...).Output()
}

func combinedOutput(ctx context.Context, directory string, args ...string) ([]byte, error) {
	return command(ctx, directory, args...).CombinedOutput()
}

func command(ctx context.Context, directory string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", directory}, args...)...)
	// Git hooks export the caller's index. Each workspace must use its own.
	cmd.Env = slices.DeleteFunc(cmd.Environ(), func(entry string) bool {
		return strings.HasPrefix(entry, "GIT_INDEX_FILE=")
	})
	return cmd
}
