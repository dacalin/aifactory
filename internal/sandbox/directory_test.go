package sandbox

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDirectoryRejectsSymlinkInWriteBoundary(t *testing.T) {
	project, outside := t.TempDir(), t.TempDir()
	if err := os.Symlink(outside, filepath.Join(project, "policy")); err != nil {
		t.Fatal(err)
	}
	if _, err := Directory(project, "policy/rules"); err == nil {
		t.Fatal("accepted a rule directory outside the project")
	}
	if entries, err := os.ReadDir(outside); err != nil || len(entries) != 0 {
		t.Fatalf("sandbox setup wrote outside the project: %v, %v", entries, err)
	}
}

func TestRuntimeDirectoryRejectsSymlink(t *testing.T) {
	rules, outside := t.TempDir(), t.TempDir()
	if err := os.Symlink(outside, filepath.Join(rules, ".tmp")); err != nil {
		t.Fatal(err)
	}
	if _, err := RuntimeDirectory(rules); err == nil {
		t.Fatal("accepted a linked runtime directory")
	}
	if entries, err := os.ReadDir(outside); err != nil || len(entries) != 0 {
		t.Fatalf("sandbox setup wrote outside the rules: %v, %v", entries, err)
	}
}
