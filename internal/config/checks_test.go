package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestCheckerCommandsLoad(t *testing.T) {
	for _, test := range []struct {
		name, yaml string
		build, run []string
	}{
		{"compiled", "checks: {build: [compiler, --output, checkrules], run: [./checkrules]}", []string{"compiler", "--output", "checkrules"}, []string{"./checkrules"}},
		{"interpreted", "checks: {run: [python3, checkrules.py]}", nil, []string{"python3", "checkrules.py"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "factory.yaml"), []byte(test.yaml), 0600); err != nil {
				t.Fatal(err)
			}
			cfg, err := Load(dir)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(cfg.Checks.Build, test.build) || !reflect.DeepEqual(cfg.Checks.Run, test.run) {
				t.Fatalf("unexpected commands: %+v", cfg.Checks)
			}
		})
	}
}

func TestInvalidCheckerCommands(t *testing.T) {
	for _, data := range []string{
		"checks: {run: 'python3 checkrules.py'}",
		"checks: {run: ['', checkrules.py]}",
		"checks: {build: [' ', checkrules]}",
		"checks: {build: 'compiler checkrules'}",
	} {
		t.Run(data, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "factory.yaml"), []byte(data), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := Load(dir); err == nil {
				t.Fatal("accepted invalid command")
			}
		})
	}
}
