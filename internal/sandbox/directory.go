package sandbox

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Directory resolves the write boundary without following links within the project.
func Directory(project, rules string) (string, error) {
	if !filepath.IsLocal(rules) || filepath.Clean(rules) == "." {
		return "", fmt.Errorf("invalid rule sandbox directory %q", rules)
	}
	project, err := filepath.EvalSymlinks(project)
	if err != nil {
		return "", err
	}
	var current string
	for _, segment := range strings.Split(filepath.Clean(rules), string(filepath.Separator)) {
		if strings.EqualFold(segment, ".git") {
			return "", fmt.Errorf("rule sandbox directory cannot be inside Git metadata")
		}
		current = filepath.Join(current, segment)
		if err := ensureDirectory(filepath.Join(project, current)); err != nil {
			return "", err
		}
	}
	return filepath.Abs(filepath.Join(project, current))
}

// RuntimeDirectory stores tool state under the same writable boundary.
func RuntimeDirectory(rules string) (string, error) {
	directory := filepath.Join(rules, ".tmp")
	root, err := os.OpenRoot(rules)
	if err != nil {
		return "", err
	}
	defer root.Close()
	if err := root.Mkdir(".tmp", 0700); err != nil && !errors.Is(err, os.ErrExist) {
		return "", err
	}
	info, err := root.Lstat(".tmp")
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("rule sandbox runtime path must be a real directory: %s", directory)
	}
	file, err := root.OpenFile(".tmp/.gitignore", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if errors.Is(err, os.ErrExist) {
		return directory, nil
	}
	if err != nil {
		return "", err
	}
	_, writeErr := file.WriteString("*\n")
	return directory, errors.Join(writeErr, file.Close())
}

func ensureDirectory(path string) error {
	if err := os.Mkdir(path, 0700); err != nil && !errors.Is(err, os.ErrExist) {
		return err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("rule sandbox directory must be a real directory: %s", path)
	}
	return nil
}
