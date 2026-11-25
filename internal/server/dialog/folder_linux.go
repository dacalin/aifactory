package dialog

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

type NativeFolderPicker struct{}

func (NativeFolderPicker) SelectFolder(ctx context.Context, title string) (string, error) {
	if _, err := exec.LookPath("zenity"); err == nil {
		return selectWith(ctx, "zenity", "--file-selection", "--directory", "--title="+title)
	}
	if _, err := exec.LookPath("kdialog"); err == nil {
		return selectWith(ctx, "kdialog", "--getexistingdirectory", ".", "--title", title)
	}
	return "", errors.New("el selector nativo en Linux necesita zenity o kdialog")
}

func selectWith(ctx context.Context, executable string, args ...string) (string, error) {
	output, err := exec.CommandContext(ctx, executable, args...).Output()
	if err == nil {
		return strings.TrimRight(string(output), "\r\n"), nil
	}
	var exitError *exec.ExitError
	if errors.As(err, &exitError) && exitError.ExitCode() == 1 {
		return "", ErrCancelled
	}
	return "", fmt.Errorf("run %s folder dialog: %w", executable, err)
}
