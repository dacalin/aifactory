package dialog

import (
	"context"
	"errors"
	"os/exec"
	"strings"
)

type NativeFolderPicker struct{}

func (NativeFolderPicker) SelectFolder(ctx context.Context, title string) (string, error) {
	title = strings.ReplaceAll(title, `\`, `\\`)
	title = strings.ReplaceAll(title, `"`, `\"`)
	script := `POSIX path of (choose folder with prompt "` + title + `")`
	command := exec.CommandContext(ctx, "osascript", "-e", script)
	output, err := command.Output()
	if err != nil {
		var exitError *exec.ExitError
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		if errors.As(err, &exitError) {
			if strings.Contains(string(exitError.Stderr), "(-128)") || strings.Contains(string(exitError.Stderr), "User canceled") {
				return "", ErrCancelled
			}
			if exitError.ExitCode() == 69 {
				return "", errors.New("macOS no ha podido abrir el selector de carpetas en esta sesión")
			}
		}
		return "", err
	}
	return strings.TrimRight(string(output), "\r\n"), nil
}
