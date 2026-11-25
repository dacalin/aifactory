package runnercli

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestRestrictedCommandSeparatesProviderHomeAndTemporaryFiles(t *testing.T) {
	project, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	bin := t.TempDir()
	for _, program := range []string{"codex", "bwrap"} {
		if err := os.WriteFile(filepath.Join(bin, program), []byte("#!/bin/sh\n"), 0700); err != nil {
			t.Fatal(err)
		}
	}
	config := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(config, nil, 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	t.Setenv("CODEX_HOME", filepath.Dir(config))

	rules := filepath.Join("policy", "rules")
	cli := NewCLI("codex", "CODEX_HOME", []string{"config.toml"}, WithWritableDirs(rules))
	cmd, err := cli.Command(t.Context(), project)
	if err != nil {
		t.Fatal(err)
	}
	runtimeDir := filepath.Join(project, rules, ".tmp")
	if filepath.Dir(cli.homeDir) != filepath.Join(runtimeDir, "home") {
		t.Fatalf("provider home = %q, want a session directory under .tmp/home", cli.homeDir)
	}
	work := filepath.Join(runtimeDir, "work", filepath.Base(cli.homeDir))
	for _, assignment := range []string{
		"CODEX_HOME=" + cli.homeDir,
		"TMPDIR=" + work,
		"GOTMPDIR=" + work,
		"GOCACHE=" + filepath.Join(work, "go-build"),
	} {
		if !slices.Contains(cmd.Environ(), assignment) {
			t.Errorf("missing %q in command environment", assignment)
		}
	}
	if target, err := os.Readlink(filepath.Join(cli.homeDir, "config.toml")); err != nil || target != config {
		t.Errorf("provider config link = %q, %v; want %q", target, err, config)
	}
	followUp, err := cli.Command(t.Context(), project)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(cmd.Environ(), followUp.Environ()) {
		t.Fatal("follow-up command did not reuse the session environment")
	}
	sibling := filepath.Join(runtimeDir, "work", "other-session")
	if err := os.Mkdir(sibling, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(sibling, filepath.Join(work, "other-session")); err != nil {
		t.Fatal(err)
	}
	if err := cli.Close(); err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{cli.homeDir, work} {
		if _, err := os.Lstat(dir); !os.IsNotExist(err) {
			t.Errorf("session path %s still exists or could not be checked: %v", dir, err)
		}
	}
	for _, path := range []string{sibling, config} {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("cleanup touched %s: %v", path, err)
		}
	}
}
