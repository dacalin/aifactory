package runnercli

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestWithWritableDirsUsesExplicitProjectSubdirectories(t *testing.T) {
	project, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	dirs := []string{"policy/rules", "reports"}
	option := WithWritableDirs(dirs...)
	dirs[1] = "../outside"
	cli := NewCLI("example", "EXAMPLE_HOME", nil, option)
	got, err := cli.boundaries(project)
	want := []string{filepath.Join(project, "policy/rules"), filepath.Join(project, "reports")}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("writable directories = %v, %v; want %v", got, err, want)
	}
}

func TestWritableDirsWithEmptyListFailsClosed(t *testing.T) {
	cli := NewCLI("example", "EXAMPLE_HOME", nil, WithWritableDirs())
	if !cli.HasWriteRestriction() {
		t.Fatal("explicit empty list disabled the write restriction")
	}
	if cmd, err := cli.Command(t.Context(), t.TempDir()); err == nil || cmd != nil {
		t.Fatalf("expected configuration error, got %v, %v", cmd, err)
	}
}
