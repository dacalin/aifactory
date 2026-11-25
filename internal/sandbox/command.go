package sandbox

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// Command runs a child whose filesystem writes are restricted by the OS.
func Command(ctx context.Context, writableDir, workingDir, executable string, args ...string) (*exec.Cmd, error) {
	return CommandWithWritableDirs(ctx, []string{writableDir}, workingDir, executable, args...)
}

// CommandWithWritableDirs restricts writes to the supplied, resolved directories.
func CommandWithWritableDirs(ctx context.Context, dirs []string, workingDir, executable string, args ...string) (*exec.Cmd, error) {
	program, err := exec.LookPath(executable)
	if err != nil {
		return nil, err
	}
	wrapper, options, err := launcher(dirs)
	if err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(ctx, wrapper, append(append(options, program), args...)...)
	cmd.Dir = workingDir
	return cmd, nil
}

func launcher(dirs []string) (string, []string, error) {
	switch runtime.GOOS {
	case "darwin":
		return "/usr/bin/sandbox-exec", macSandboxArgs(dirs), nil
	case "linux":
		path, err := exec.LookPath("bwrap")
		if err != nil {
			return "", nil, fmt.Errorf("rule write sandbox requires bubblewrap (bwrap): %w", err)
		}
		return path, linuxSandboxArgs(dirs), nil
	default:
		return "", nil, fmt.Errorf("rule write sandbox is unavailable on %s", runtime.GOOS)
	}
}

func macSandboxArgs(dirs []string) []string {
	var options []string
	var profile strings.Builder
	profile.WriteString("(version 1)\n(allow default)\n(deny file-write*)\n")
	for i, dir := range dirs {
		name := fmt.Sprintf("WRITE_DIR_%d", i)
		options = append(options, "-D", name+"="+dir)
		fmt.Fprintf(&profile, "(allow file-write* (subpath (param %q)))\n", name)
	}
	profile.WriteString("(allow file-write-data (literal \"/dev/null\"))\n")
	return append(options, "-p", profile.String())
}

func linuxSandboxArgs(dirs []string) []string {
	options := []string{
		"--unshare-user", "--disable-userns", "--unshare-pid", "--die-with-parent",
		"--ro-bind", "/", "/", "--dev", "/dev", "--remount-ro", "/dev", "--proc", "/proc",
	}
	for _, dir := range dirs {
		options = append(options, "--bind", dir, dir)
	}
	return append(options, "--")
}
