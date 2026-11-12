package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fixtureRepository(t *testing.T, files map[string]string) *repository {
	t.Helper()
	root := t.TempDir()
	files["go.mod"] = "module example.com/fixture\n\ngo 1.25.0\n"
	for name, data := range files {
		destination := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(destination), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(destination, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	repo, err := newRepository(root, ".ai-factory", ".ai-factory/rules")
	if err != nil {
		t.Fatal(err)
	}
	return repo
}

func TestRunAllContinuesAfterFailure(t *testing.T) {
	original := checks
	defer func() { checks = original }()
	laterRan := false
	checks = map[string]checkFunc{
		"fails": func(*repository) error { return errors.New("expected failure") },
		"later": func(*repository) error { laterRan = true; return nil },
	}
	var output bytes.Buffer
	err := runChecks(&repository{rulesDir: "rules"}, []string{"fails", "later"}, &output)
	if err == nil || !laterRan || !strings.Contains(output.String(), "PASS later") {
		t.Fatalf("failure continuation did not work: %v, %s", err, &output)
	}
}

func TestInterfaceFileContents(t *testing.T) {
	cases := []struct {
		name, declarations string
		invalid            bool
	}{
		{"supporting types", "type Handler interface{ Handle(Callback) }; type Callback func(Request); type Request string", false},
		{"ordinary file", "type Payload struct{}; func Handle() {}", false},
		{"struct", "type Handler interface{ Handle() }; type Payload struct{}", true},
		{"function", "type Handler interface{ Handle() }; func Handle() {}", true},
		{"constant", "type Handler interface{ Handle() }; const Value = 1", true},
		{"variable", "type Handler interface{ Handle() }; var Value = 1", true},
		{"unused type", "type Handler interface{ Handle() }; type Extra string", true},
		{"method-name collision", "type Handler interface{ Extra() }; type Extra string", true},
		{"nested struct", "type Handler interface{ Handle(Payload) }; type Payload []struct{ Value int }", true},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			repo := fixtureRepository(t, map[string]string{"contracts_test.go": "package example\n" + test.declarations})
			err := checkInterfaceFiles(repo)
			if (err != nil) != test.invalid {
				t.Fatalf("check returned %v", err)
			}
		})
	}
}

func TestInterfaceTypeNames(t *testing.T) {
	original := interfaceNameSettings
	defer func() { interfaceNameSettings = original }()
	interfaceNameSettings = struct {
		Roots              []string
		ExcludedRoots      []string
		ForbiddenFragments []string
	}{Roots: []string{"."}, ForbiddenFragments: []string{"interface"}}

	for _, test := range []struct {
		name    string
		files   map[string]string
		invalid bool
	}{
		{"role name", map[string]string{"runner.go": "package example\n\ntype AgentRunner interface{}\n"}, false},
		{"forbidden suffix", map[string]string{"runner.go": "package example\n\ntype AgentRunnerInterface interface{}\n"}, true},
		{"case insensitive fragment", map[string]string{"runner_test.go": "package example\n\ntype testinterface interface{}\n"}, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			repo := fixtureRepository(t, test.files)
			err := checkInterfaceTypeNames(repo)
			if (err != nil) != test.invalid {
				t.Fatalf("check returned %v", err)
			}
		})
	}
}

func TestInterfaceTypeNamesRespectsConfiguredRootsAndExclusions(t *testing.T) {
	original := interfaceNameSettings
	defer func() { interfaceNameSettings = original }()
	interfaceNameSettings = struct {
		Roots              []string
		ExcludedRoots      []string
		ForbiddenFragments []string
	}{
		Roots:              []string{"selected"},
		ExcludedRoots:      []string{"selected/legacy"},
		ForbiddenFragments: []string{"interface"},
	}
	repo := fixtureRepository(t, map[string]string{
		"selected/runner.go":        "package selected\n\ntype AgentRunner interface{}\n",
		"selected/legacy/runner.go": "package legacy\n\ntype LegacyInterface interface{}\n",
		"outside/runner.go":         "package outside\n\ntype OutsideInterface interface{}\n",
	})
	if err := checkInterfaceTypeNames(repo); err != nil {
		t.Fatalf("check returned %v", err)
	}
}

func TestReadabilityThresholds(t *testing.T) {
	for _, test := range []struct {
		lines, function, complexity int
		invalid                     bool
	}{
		{300, 60, 15, false}, {301, 60, 15, true}, {300, 61, 15, true}, {300, 60, 16, true},
	} {
		t.Run(fmt.Sprintf("%d-%d-%d", test.lines, test.function, test.complexity), func(t *testing.T) {
			source := []string{"package example", "", "func Run(flag bool) {"}
			for i := 0; i < test.complexity; i++ {
				source = append(source, "if flag {}")
			}
			for len(source) < test.function+1 {
				source = append(source, "// padding")
			}
			source = append(source, "}")
			for len(source) < test.lines {
				source = append(source, "// padding")
			}
			repo := fixtureRepository(t, map[string]string{"source.go": strings.Join(source, "\n") + "\n"})
			err := checkReadability(repo)
			if (err != nil) != test.invalid {
				t.Fatalf("check returned %v", err)
			}
		})
	}
}

func TestRootMetadataDirectoryDoesNotHideApplication(t *testing.T) {
	repo := fixtureRepository(t, map[string]string{
		"app/source.go":    "package app\n",
		"checks/broken.go": "invalid source",
		".tmp/broken.go":   "invalid source",
	})
	repo.dataDir, repo.rulesDir = ".", "checks"
	files, err := repo.sources(false)
	if err != nil || len(files) != 1 || files[0].name != "app/source.go" {
		t.Fatalf("unexpected source scan %v, %v", files, err)
	}
}

func TestCheckerDirectories(t *testing.T) {
	for _, test := range []struct {
		name, yaml, data, rules, wantData, wantRules string
	}{
		{"default", "", "", "", ".ai-factory", ".ai-factory/rules"},
		{"custom", "dir: generated\nrule-agent:\n  dir: checks\n", "", "", "generated", "generated/checks"},
		{"override", "dir: generated\nrule-agent:\n  dir: checks\n", "alternate", "", "alternate", "alternate/checks"},
		{"legacy", "rules_dir: checks/rules\n", "", "", "checks", "checks/rules"},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			if test.yaml != "" {
				if err := os.WriteFile(filepath.Join(root, "factory.yaml"), []byte(test.yaml), 0600); err != nil {
					t.Fatal(err)
				}
			}
			data, rules, err := checkerDirectories(root, test.data, test.rules)
			if err != nil || data != filepath.FromSlash(test.wantData) || rules != filepath.FromSlash(test.wantRules) {
				t.Fatalf("directories = %q, %q, %v", data, rules, err)
			}
		})
	}
}
