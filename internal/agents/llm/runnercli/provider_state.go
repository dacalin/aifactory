package runnercli

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/dacalin/aifactory/internal/sandbox"
)

func providerState(boundary, provider, homeVariable string, inputs []string) (string, string, func() error, error) {
	runtimeDir, err := sandbox.RuntimeDirectory(boundary)
	if err != nil {
		return "", "", nil, err
	}
	homes, err := sandbox.Directory(runtimeDir, "home")
	if err != nil {
		return "", "", nil, err
	}
	work, err := sandbox.Directory(runtimeDir, "work")
	if err != nil {
		return "", "", nil, err
	}
	home, err := os.MkdirTemp(homes, provider+"-")
	if err != nil {
		return "", "", nil, err
	}
	if err := linkProviderInputs(provider, homeVariable, home, inputs); err != nil {
		return "", "", nil, errors.Join(err, os.RemoveAll(home))
	}
	// Codex refuses to create helper binaries when its home is inside TMPDIR.
	temporary := filepath.Join(work, filepath.Base(home))
	if err := os.Mkdir(temporary, 0700); err != nil {
		return "", "", nil, errors.Join(err, os.RemoveAll(home))
	}
	cleanup, err := sessionCleanup(home, temporary)
	if err != nil {
		return "", "", nil, errors.Join(err, os.RemoveAll(home), os.RemoveAll(temporary))
	}
	return home, temporary, cleanup, nil
}

func sessionCleanup(home, temporary string) (func() error, error) {
	homes, err := os.OpenRoot(filepath.Dir(home))
	if err != nil {
		return nil, err
	}
	work, err := os.OpenRoot(filepath.Dir(temporary))
	if err != nil {
		return nil, errors.Join(err, homes.Close())
	}
	return func() error {
		return errors.Join(homes.RemoveAll(filepath.Base(home)), work.RemoveAll(filepath.Base(temporary)),
			homes.Close(), work.Close())
	}, nil
}

func linkProviderInputs(provider, homeVariable, state string, inputs []string) error {
	home := os.Getenv(homeVariable)
	if home == "" {
		userHome, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		home = filepath.Join(userHome, "."+provider)
	}
	home, err := filepath.Abs(home)
	if err != nil {
		return err
	}
	for _, name := range inputs {
		if err := linkExistingInput(filepath.Join(home, name), filepath.Join(state, name)); err != nil {
			return err
		}
	}
	return nil
}

func linkExistingInput(source, target string) error {
	if _, err := os.Stat(source); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	return os.Symlink(source, target)
}
