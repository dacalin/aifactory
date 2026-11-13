package commands

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestReviewExecutesConfiguredInterpreters(t *testing.T) {
	for _, fixture := range []struct{ runtime, name, source string }{
		{"python3", "check rules.py", `import os,sys
print("ROOT:"+os.environ["AIFACTORY_PROJECT_DIR"])
print("ARGS:"+"|".join(sys.argv[1:]))
print("CWD:"+os.getcwd())
print("PASS python")
`},
		{"node", "check rules.js", `console.log("ROOT:"+process.env.AIFACTORY_PROJECT_DIR);
console.log("ARGS:"+process.argv.slice(2).join("|"));
console.log("CWD:"+process.cwd());
console.log("PASS node");
`},
	} {
		t.Run(fixture.runtime, func(t *testing.T) {
			if _, err := exec.LookPath(fixture.runtime); err != nil {
				t.Skip(fixture.runtime + " is unavailable")
			}
			root := t.TempDir()
			t.Chdir(root)
			settings := "dir: policy\nrule-agent:\n  dir: checks\nchecks:\n  run: [" + fixture.runtime + ", '" + fixture.name + "', --mode, 'strict mode']\n"
			writeReviewFixture(t, root, "factory.yaml", settings, 0600)
			writeReviewFixture(t, root, filepath.Join("policy/checks", fixture.name), fixture.source, 0600)
			// A module file must not trigger a language-specific, implicit build.
			writeReviewFixture(t, root, "policy/checks/go.mod", "invalid go module", 0600)
			output, diagnostics, err := executeReview(t, "check with spaces & symbols")
			if err != nil {
				t.Fatalf("interpreter: %s, %v", diagnostics, err)
			}
			for _, expected := range []string{"ROOT:" + root, "CWD:" + filepath.Join(root, "policy/checks"), "ARGS:--mode|strict mode|check with spaces & symbols", "PASS"} {
				if !strings.Contains(output, expected) {
					t.Fatalf("missing %q: %s", expected, output)
				}
			}
		})
	}
}
