package commands

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dacalin/aifactory/internal/agents"
	"github.com/dacalin/aifactory/internal/sandbox"
)

func requireReviewSandbox(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	cmd, err := sandbox.Command(context.Background(), dir, dir, "/bin/sh", "-c", ":")
	if err != nil {
		if strings.Contains(err.Error(), "requires bubblewrap") {
			t.Skip(err)
		}
		t.Fatal(err)
	}
	if output, err := cmd.CombinedOutput(); err != nil {
		message := string(output)
		if strings.Contains(message, "sandbox_apply: Operation not permitted") ||
			strings.Contains(message, "No permissions to create a new namespace") ||
			strings.Contains(message, "Creating new namespace failed: Operation not permitted") {
			t.Skip(strings.TrimSpace(message))
		}
		t.Fatalf("start rule sandbox: %v: %s", err, output)
	}
}

func writeReviewFixture(t *testing.T, root, name, content string, mode os.FileMode) {
	t.Helper()
	name = filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(name), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(name, []byte(content), mode); err != nil {
		t.Fatal(err)
	}
}

func executeReview(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	command := NewRootCommand(func() (agents.Agent, error) { t.Fatal("review constructed an LLM agent"); return nil, nil }, nil)
	var output, diagnostics bytes.Buffer
	command.SetOut(&output)
	command.SetErr(&diagnostics)
	command.SetArgs(append([]string{"rule", "review"}, args...))
	err := command.Execute()
	return output.String(), diagnostics.String(), err
}

func TestReviewUsesSharedEntryPointInCurrentWorktree(t *testing.T) {
	requireReviewSandbox(t)
	root := t.TempDir()
	writeReviewFixture(t, root, "factory.yaml", "dir: policy\nrule-agent:\n  dir: checks\nchecks:\n  run: [sh, audit.sh]\n", 0600)
	writeReviewFixture(t, root, "policy/checks/audit.sh", `#!/bin/sh
printf 'ROOT:%s\n' "$AIFACTORY_PROJECT_DIR"
case "$1" in
  all) printf 'PASS first\nPASS second\nAll 2 checks passed.\n' ;;
  broken) printf 'FAIL broken\nsource.go:4: violation\nPASS later\n'; printf '1 of 2 checks failed\n' >&2; exit 1 ;;
  --list) printf 'first\nsecond\n' ;;
  *) printf 'unknown check\n' >&2; exit 2 ;;
esac
`, 0700)
	writeReviewFixture(t, root, "nested/.keep", "", 0600)
	t.Chdir(filepath.Join(root, "nested"))
	for _, test := range []struct {
		args   []string
		text   string
		failed bool
	}{
		{nil, "All 2 checks passed.", false},
		{[]string{"all"}, "PASS first", false},
		{[]string{"broken"}, "source.go:4: violation\nPASS later", true},
		{[]string{"--list"}, "first\nsecond", false},
		{[]string{"unknown"}, "", true},
	} {
		output, diagnostics, err := executeReview(t, test.args...)
		if (err != nil) != test.failed || !strings.Contains(output, test.text) || !strings.Contains(output, "ROOT:"+root) {
			t.Fatalf("review %v: %q, %q, %v", test.args, output, diagnostics, err)
		}
		if len(test.args) > 0 && test.args[0] == "broken" && !strings.Contains(diagnostics, "1 of 2 checks failed") {
			t.Fatal("lost failure diagnostics")
		}
	}
	for _, args := range [][]string{{"first", "second"}, {"--list", "first"}} {
		if _, _, err := executeReview(t, args...); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
}

func TestReviewRebuildsGoRunnerAndRejectsBrokenBuild(t *testing.T) {
	requireReviewSandbox(t)
	root := t.TempDir()
	t.Chdir(root)
	writeReviewFixture(t, root, "factory.yaml", "dir: policy\nrule-agent:\n  dir: checks\nchecks:\n  build: [go, build, -o, verify, .]\n  run: [./verify]\n", 0600)
	writeReviewFixture(t, root, "policy/checks/go.mod", "module example.com/checks\n\ngo 1.25.0\n", 0600)
	writeReviewFixture(t, root, "policy/checks/main.go", "package main\nfunc main() { println(\"STALE\") }\n", 0600)
	build := exec.Command("go", "build", "-o", "verify", ".")
	build.Dir = filepath.Join(root, "policy/checks")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build stale fixture: %s, %v", output, err)
	}
	writeReviewFixture(t, root, "policy/checks/main.go", "package main\nimport (\"fmt\"; \"os\")\nfunc main() { fmt.Println(\"PASS current-source\", os.Args[1]) }\n", 0600)
	output, diagnostics, err := executeReview(t)
	if err != nil || !strings.Contains(output, "PASS current-source all") || strings.Contains(output, "STALE") {
		t.Fatalf("rebuild: %q, %q, %v", output, diagnostics, err)
	}
	writeReviewFixture(t, root, "policy/checks/main.go", "not valid go", 0600)
	output, _, err = executeReview(t)
	if err == nil || !strings.Contains(err.Error(), "build configured checker") || strings.Contains(output, "PASS") {
		t.Fatalf("ran stale binary after build failure: %q, %v", output, err)
	}
	entries, err := filepath.Glob(filepath.Join(root, "policy/checks/.tmp/review-*"))
	if err != nil || len(entries) != 0 {
		t.Fatalf("temporary build artifacts remain: %v, %v", entries, err)
	}
}

func TestReviewSandboxRestrictsCheckerWrites(t *testing.T) {
	requireReviewSandbox(t)
	root := t.TempDir()
	t.Chdir(root)
	writeReviewFixture(t, root, "factory.yaml", "dir: policy\nrule-agent:\n  dir: checks\nchecks:\n  run: [sh, audit.sh]\n", 0600)
	writeReviewFixture(t, root, "source.go", "untouched", 0600)
	writeReviewFixture(t, root, "policy/checks/audit.sh", `#!/bin/sh
set -eu
test "$(cat "$AIFACTORY_PROJECT_DIR/source.go")" = untouched
echo pass > allowed
echo temp > "$TMPDIR/artifact"
if sh -c 'echo forbidden > "$1"' sh "$AIFACTORY_PROJECT_DIR/source.go"; then exit 1; fi
if sh -c 'echo forbidden > "$1"' sh source-link; then exit 1; fi
if sh -c 'echo forbidden > "$1"' sh ../outside; then exit 1; fi
`, 0600)
	rules := filepath.Join(root, "policy/checks")
	if err := os.Symlink(filepath.Join(root, "source.go"), filepath.Join(rules, "source-link")); err != nil {
		t.Fatal(err)
	}
	if _, diagnostics, err := executeReview(t); err != nil {
		t.Fatalf("review failed: %v: %s", err, diagnostics)
	}
	if content, err := os.ReadFile(filepath.Join(root, "source.go")); err != nil || string(content) != "untouched" {
		t.Fatalf("source changed: %q, %v", content, err)
	}
	if content, err := os.ReadFile(filepath.Join(rules, "allowed")); err != nil || string(content) != "pass\n" {
		t.Fatalf("expected rule output: %q, %v", content, err)
	}
}

func TestReviewDoesNotEscapeWorktree(t *testing.T) {
	root := t.TempDir()
	writeReviewFixture(t, root, "factory.yaml", "dir: policy\n", 0600)
	writeReviewFixture(t, root, "worktree/.git", "gitdir: somewhere\n", 0600)
	writeReviewFixture(t, root, "worktree/nested/.keep", "", 0600)
	t.Chdir(filepath.Join(root, "worktree/nested"))
	if _, _, err := executeReview(t); err == nil || !strings.Contains(err.Error(), "factory.yaml not found") {
		t.Fatalf("escaped worktree: %v", err)
	}
}

func TestReviewMissingRunnerAndHelp(t *testing.T) {
	t.Chdir(t.TempDir())
	output, _, err := executeReview(t, "--help")
	if err != nil || !strings.Contains(output, "review [check-id|all]") {
		t.Fatalf("help: %q, %v", output, err)
	}
	if err := os.WriteFile("factory.yaml", []byte("dir: policy\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := executeReview(t); err == nil || !strings.Contains(err.Error(), "checks.run is not configured") {
		t.Fatalf("missing runner: %v", err)
	}
}
