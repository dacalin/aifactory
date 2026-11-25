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

func TestFindRootUsesManagedProjectDirectory(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "factory.yaml"), []byte("dir: .ai-factory\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AIFACTORY_PROJECT_DIR", root)
	found, err := findRoot("")
	if err != nil || found != root {
		t.Fatalf("findRoot() = %q, %v; want %q", found, err, root)
	}
}

func TestStructMethodsAndContracts(t *testing.T) {
	original := structFileSettings
	defer func() { structFileSettings = original }()
	structFileSettings = struct {
		SourceRoots   []string
		ExcludedRoots []string
	}{SourceRoots: []string{"."}}

	for _, test := range []struct {
		name     string
		files    map[string]string
		invalid  bool
		contains string
	}{
		{
			name: "co-located method and assertion",
			files: map[string]string{
				"contract.go": "package example\n\ntype Worker interface { Work() }\n",
				"manager.go":  "package example\n\ntype Manager struct{}\n\nvar _ Worker = (*Manager)(nil)\n\nfunc (Manager) Work() {}\n",
			},
		},
		{
			name: "platform-specific declarations",
			files: map[string]string{
				"manager_darwin.go": "package example\n\ntype Manager struct{}\n\nfunc (Manager) Work() {}\n",
				"manager_linux.go":  "package example\n\ntype Manager struct{}\n\nfunc (Manager) Work() {}\n",
			},
		},
		{
			name: "method in another file",
			files: map[string]string{
				"manager.go": "package example\n\ntype Manager struct{}\n",
				"methods.go": "package example\n\nfunc (Manager) Work() {}\n",
			},
			invalid:  true,
			contains: "must be declared in manager.go",
		},
		{
			name: "method before struct",
			files: map[string]string{
				"manager.go": "package example\n\nfunc (Manager) Work() {}\n\ntype Manager struct{}\n",
			},
			invalid:  true,
			contains: "must follow",
		},
		{
			name: "assertion in another file",
			files: map[string]string{
				"contract.go":  "package example\n\ntype Worker interface { Work() }\n",
				"manager.go":   "package example\n\ntype Manager struct{}\n\nfunc (Manager) Work() {}\n",
				"assertion.go": "package example\n\nvar _ Worker = (*Manager)(nil)\n",
			},
			invalid:  true,
			contains: "conformance assertion",
		},
		{
			name: "qualified assertion in another package file",
			files: map[string]string{
				"storage/database.go":  "package storage\n\ntype Database interface{}\n",
				"storage/open.go":      "package storage\n\nimport \"example.com/fixture/storage/sqlite\"\n\nvar _ Database = (*sqlite.DB)(nil)\n",
				"storage/sqlite/db.go": "package sqlite\n\ntype DB struct{}\n",
			},
			invalid:  true,
			contains: "storage/sqlite/db.go",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := checkStructMethodsAndContracts(fixtureRepository(t, test.files))
			if (err != nil) != test.invalid {
				t.Fatalf("check returned %v", err)
			}
			if test.contains != "" && (err == nil || !strings.Contains(err.Error(), test.contains)) {
				t.Fatalf("check error %q does not contain %q", err, test.contains)
			}
		})
	}
}

func TestStructMethodsAndContractsRespectsConfiguredRootsAndExclusions(t *testing.T) {
	original := structFileSettings
	defer func() { structFileSettings = original }()
	structFileSettings = struct {
		SourceRoots   []string
		ExcludedRoots []string
	}{
		SourceRoots:   []string{"selected"},
		ExcludedRoots: []string{"selected/legacy"},
	}
	allowed := map[string]string{
		"selected/manager.go":        "package selected\n\ntype Manager struct{}\n\nfunc (Manager) Run() {}\n",
		"selected/legacy/manager.go": "package legacy\n\ntype Legacy struct{}\n",
		"selected/legacy/methods.go": "package legacy\n\nfunc (Legacy) Run() {}\n",
		"outside/manager.go":         "package outside\n\ntype Outside struct{}\n",
		"outside/methods.go":         "package outside\n\nfunc (Outside) Run() {}\n",
	}
	if err := checkStructMethodsAndContracts(fixtureRepository(t, allowed)); err != nil {
		t.Fatalf("configured scope returned %v", err)
	}
	violating := mergeFixtureFiles(allowed, map[string]string{
		"selected/methods.go": "package selected\n\nfunc (Manager) Stop() {}\n",
	})
	if err := checkStructMethodsAndContracts(fixtureRepository(t, violating)); err == nil {
		t.Fatal("method outside its struct file passed in configured scope")
	}
}

func TestDatabaseAccessStaysInStorage(t *testing.T) {
	original := databaseAccessSettings
	defer func() { databaseAccessSettings = original }()
	databaseAccessSettings = databaseAccessConfiguration{
		StorageRoot:   "internal/storage",
		SourceRoots:   []string{"."},
		ExcludedRoots: []string{},
		DatabaseImportRoots: []string{
			"database/sql",
			"modernc.org/sqlite",
			"github.com/golang-migrate/migrate/v4",
		},
		AdapterRoots:       []string{"internal/storage/sqlite"},
		CompositionFiles:   []string{"bootstrap.go"},
		LifecycleInterface: "Database",
		AdapterContracts: []databaseAdapterContract{
			{Root: "internal/storage/sqlite", Struct: "DB"},
		},
	}

	boundary := map[string]string{
		"internal/storage/database.go":  "package storage\n\ntype Database interface { Close() error }\n",
		"internal/storage/sqlite/db.go": "package sqlite\n\nimport (\n \"database/sql\"\n \"example.com/fixture/internal/storage\"\n _ \"modernc.org/sqlite\"\n)\n\ntype DB struct { *sql.DB }\n\nvar _ storage.Database = (*DB)(nil)\n",
		"bootstrap.go":                  "package main\n\nimport _ \"example.com/fixture/internal/storage/sqlite\"\n",
	}
	for _, test := range []struct {
		name     string
		files    map[string]string
		invalid  bool
		contains string
	}{
		{
			name:  "database packages inside storage",
			files: boundary,
		},
		{
			name: "database package outside storage",
			files: mergeFixtureFiles(boundary, map[string]string{
				"internal/project/run.go": "package project\n\nimport \"database/sql\"\n\nvar _ sql.DB\n",
			}),
			invalid: true,
		},
		{
			name: "database package in test outside storage",
			files: mergeFixtureFiles(boundary, map[string]string{
				"bootstrap_test.go": "package main\n\nimport \"database/sql\"\n\nvar _ sql.DB\n",
			}),
			invalid: true,
		},
		{
			name: "concrete adapter outside composition",
			files: mergeFixtureFiles(boundary, map[string]string{
				"internal/project/run.go": "package project\n\nimport _ \"example.com/fixture/internal/storage/sqlite\"\n",
			}),
			invalid:  true,
			contains: "configured composition files",
		},
		{
			name: "missing adapter contract assertion",
			files: mergeFixtureFiles(boundary, map[string]string{
				"internal/storage/sqlite/db.go": "package sqlite\n\ntype DB struct{}\n",
			}),
			invalid:  true,
			contains: "must assert",
		},
		{
			name: "invalid lifecycle abstraction",
			files: map[string]string{
				"internal/storage/database.go":  "package storage\n\ntype Database interface { Close(string) error }\n",
				"internal/storage/sqlite/db.go": "package sqlite\n\nimport \"example.com/fixture/internal/storage\"\n\ntype DB struct{}\n\nvar _ storage.Database = (*DB)(nil)\n",
				"bootstrap.go":                  "package main\n\nimport _ \"example.com/fixture/internal/storage/sqlite\"\n",
			},
			invalid:  true,
			contains: "Database must declare Close() error",
		},
		{
			name: "multiple direct imports outside storage",
			files: mergeFixtureFiles(boundary, map[string]string{
				"internal/project/run.go": "package project\n\nimport (\n _ \"example.com/fixture/internal/storage/sqlite\"\n _ \"modernc.org/sqlite\"\n)\n",
			}),
			invalid:  true,
			contains: "configured composition files",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := checkDatabaseAccess(fixtureRepository(t, test.files))
			if (err != nil) != test.invalid {
				t.Fatalf("check returned %v", err)
			}
			if test.contains != "" && (err == nil || !strings.Contains(err.Error(), test.contains)) {
				t.Fatalf("check error %q does not contain %q", err, test.contains)
			}
		})
	}
}

func TestDatabaseAccessRespectsConfiguredRootsAndExclusions(t *testing.T) {
	original := databaseAccessSettings
	defer func() { databaseAccessSettings = original }()
	databaseAccessSettings = databaseAccessConfiguration{
		StorageRoot:         "persistence",
		SourceRoots:         []string{"."},
		ExcludedRoots:       []string{"legacy"},
		DatabaseImportRoots: []string{"example.com/database"},
		AdapterRoots:        []string{"persistence/custom"},
		CompositionFiles:    []string{"application/bootstrap.go"},
		LifecycleInterface:  "Connection",
		AdapterContracts:    []databaseAdapterContract{{Root: "persistence/custom", Struct: "Driver"}},
	}
	boundary := map[string]string{
		"persistence/connection.go": "package persistence\n\ntype Connection interface { Close() error }\n",
		"persistence/custom/db.go":  "package custom\n\nimport (\n _ \"example.com/database\"\n \"example.com/fixture/persistence\"\n)\n\ntype Driver struct{}\n\nvar _ persistence.Connection = (*Driver)(nil)\n",
		"application/bootstrap.go":  "package application\n\nimport _ \"example.com/fixture/persistence/custom\"\n",
		"legacy/db.go":              "package legacy\n\nimport _ \"example.com/database\"\n",
	}
	if err := checkDatabaseAccess(fixtureRepository(t, boundary)); err != nil {
		t.Fatalf("configured boundary returned %v", err)
	}
	violating := mergeFixtureFiles(boundary, map[string]string{
		"application/run.go": "package application\n\nimport _ \"example.com/fixture/persistence/custom\"\n",
	})
	if err := checkDatabaseAccess(fixtureRepository(t, violating)); err == nil {
		t.Fatal("concrete adapter import outside configured persistence root passed")
	}
}

func mergeFixtureFiles(base, additions map[string]string) map[string]string {
	merged := make(map[string]string, len(base)+len(additions))
	for name, source := range base {
		merged[name] = source
	}
	for name, source := range additions {
		merged[name] = source
	}
	return merged
}

func TestGitExecutionStaysInWorkspaceGit(t *testing.T) {
	original := gitExecutionSettings
	defer func() { gitExecutionSettings = original }()
	gitExecutionSettings = gitExecutionConfiguration{
		ExecutionRoot:   "internal/workspace/git",
		SourceRoots:     []string{"."},
		ExcludedRoots:   []string{},
		ExecutableNames: []string{"git", "git.exe"},
	}

	for _, test := range []struct {
		name     string
		files    map[string]string
		invalid  bool
		contains string
	}{
		{
			name: "Git execution inside boundary",
			files: map[string]string{
				"internal/workspace/git/worktree.go":    "package git\n\nimport runner \"os/exec\"\n\nfunc run(ctx any) { _ = runner.CommandContext(ctx, \"git\", \"status\") }\n",
				"internal/workspace/git/lookup_test.go": "package git\n\nimport . \"os/exec\"\n\nfunc lookup() { _, _ = LookPath(\"/usr/bin/git\") }\n",
			},
		},
		{
			name: "Git execution outside boundary",
			files: map[string]string{
				"internal/project/git/repository.go": "package git\n\nimport \"os/exec\"\n\nfunc run() { _ = exec.Command(\"git\", \"status\") }\n",
			},
			invalid:  true,
			contains: "internal/project/git/repository.go",
		},
		{
			name: "Git test helper outside boundary",
			files: map[string]string{
				"internal/project/git/repository_test.go": "package git\n\nimport \"os/exec\"\n\nfunc run() { _ = exec.Command(\"git.exe\", \"status\") }\n",
			},
			invalid:  true,
			contains: "repository_test.go",
		},
		{
			name: "other process outside boundary",
			files: map[string]string{
				"internal/project/open.go": "package project\n\nimport \"os/exec\"\n\nfunc open() { _ = exec.Command(\"open\", \"https://example.com\") }\n",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := checkGitExecution(fixtureRepository(t, test.files))
			if (err != nil) != test.invalid {
				t.Fatalf("check returned %v", err)
			}
			if test.contains != "" && (err == nil || !strings.Contains(err.Error(), test.contains)) {
				t.Fatalf("check error %q does not contain %q", err, test.contains)
			}
		})
	}
}

func TestGitExecutionRespectsConfiguredRootsAndExclusions(t *testing.T) {
	original := gitExecutionSettings
	defer func() { gitExecutionSettings = original }()
	gitExecutionSettings = gitExecutionConfiguration{
		ExecutionRoot:   "application/vcs",
		SourceRoots:     []string{"application"},
		ExcludedRoots:   []string{"application/legacy"},
		ExecutableNames: []string{"git-client"},
	}

	allowed := map[string]string{
		"application/vcs/run.go":    "package vcs\n\nimport \"os/exec\"\n\nfunc run() { _ = exec.Command(\"git-client\") }\n",
		"application/legacy/old.go": "package legacy\n\nimport \"os/exec\"\n\nfunc run() { _ = exec.Command(\"git-client\") }\n",
		"outside/not_scanned.go":    "package outside\n\nimport \"os/exec\"\n\nfunc run() { _ = exec.Command(\"git-client\") }\n",
	}
	if err := checkGitExecution(fixtureRepository(t, allowed)); err != nil {
		t.Fatalf("configured boundary returned %v", err)
	}
	violating := mergeFixtureFiles(allowed, map[string]string{
		"application/project/run.go": "package project\n\nimport \"os/exec\"\n\nfunc run() { _ = exec.Command(\"git-client\") }\n",
	})
	if err := checkGitExecution(fixtureRepository(t, violating)); err == nil {
		t.Fatal("Git execution outside the configured boundary passed")
	}
}

func TestFocusedDeclarationsUseAgentProviderLayout(t *testing.T) {
	allowed := agentProviderLayoutFiles()
	allowed["internal/agents/llm/runnercli/contracts.go"] = "package runnercli\n\ntype First interface{}\ntype Second interface{}\n"
	if err := checkFocusedDeclarations(fixtureRepository(t, allowed)); err != nil {
		t.Fatalf("current package layout returned %v", err)
	}
	violating := mergeFixtureFiles(agentProviderLayoutFiles(), map[string]string{
		"internal/agents/llm/codex/codex_cli_runner.go": "package codex\n\ntype CodexCLIRunner struct{}\ntype Extra struct{}\n",
	})
	if err := checkFocusedDeclarations(fixtureRepository(t, violating)); err == nil || !strings.Contains(err.Error(), "internal/agents/llm/codex") {
		t.Fatalf("multiple provider structs returned %v", err)
	}
}

func TestDependencyDirectionMatchesAgentProviderLayout(t *testing.T) {
	for _, test := range []struct {
		name     string
		files    func() map[string]string
		invalid  bool
		contains string
	}{
		{
			name: "provider may use shared CLI helper",
			files: func() map[string]string {
				files := agentProviderLayoutFiles()
				files["internal/commands/root.go"] = "package commands\n"
				return files
			},
		},
		{
			name: "agent contract package imports provider helper",
			files: func() map[string]string {
				files := agentProviderLayoutFiles()
				files["internal/agents/agent.go"] = "package agents\n\nimport _ \"example.com/fixture/internal/agents/llm/runnercli\"\n\ntype Agent interface{}\n"
				return files
			},
			invalid:  true,
			contains: "must not import",
		},
		{
			name: "provider imports CLI commands",
			files: func() map[string]string {
				files := agentProviderLayoutFiles()
				files["internal/commands/root.go"] = "package commands\n"
				files["internal/agents/llm/codex/codex_cli_runner.go"] = `package codex

import (
	"example.com/fixture/internal/agents"
	_ "example.com/fixture/internal/commands"
	"example.com/fixture/internal/agents/llm/runnercli"
)

type CodexCLIRunner struct { cli *runnercli.CLI }

var _ agents.AgentRunner = (*CodexCLIRunner)(nil)
`
				return files
			},
			invalid:  true,
			contains: "must not import",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := checkDependencies(fixtureRepository(t, test.files()))
			if (err != nil) != test.invalid {
				t.Fatalf("check returned %v", err)
			}
			if test.contains != "" && (err == nil || !strings.Contains(err.Error(), test.contains)) {
				t.Fatalf("check error %q does not contain %q", err, test.contains)
			}
		})
	}
}

func TestModelAccessUsesProviderRoot(t *testing.T) {
	for _, test := range []struct {
		name     string
		files    map[string]string
		invalid  bool
		contains string
	}{
		{
			name: "Codex runner under provider root with conformance assertion",
			files: map[string]string{
				"internal/agents/agent_runner.go": "package agents\n\ntype AgentRunner interface{}\n",
				"internal/agents/llm/codex/codex_cli_runner.go": `package codex

import (
	"os/exec"

	"example.com/fixture/internal/agents"
)

type CodexCLIRunner struct{}

var _ agents.AgentRunner = (*CodexCLIRunner)(nil)

func command() { _ = exec.Command("codex") }
`,
			},
		},
		{
			name: "provider access outside provider root",
			files: map[string]string{
				"internal/project/codex.go": "package project\n\nimport \"os/exec\"\n\nfunc command() { _ = exec.Command(\"codex\") }\n",
			},
			invalid:  true,
			contains: "model access belongs under internal/agents/llm",
		},
		{
			name: "provider access without runner conformance",
			files: map[string]string{
				"internal/agents/llm/codex/codex_cli_runner.go": "package codex\n\nimport \"os/exec\"\n\ntype CodexCLIRunner struct{}\n\nfunc command() { _ = exec.Command(\"codex\") }\n",
			},
			invalid:  true,
			contains: "agents.AgentRunner conformance assertion",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := checkModelAccess(fixtureRepository(t, test.files))
			if (err != nil) != test.invalid {
				t.Fatalf("check returned %v", err)
			}
			if test.contains != "" && (err == nil || !strings.Contains(err.Error(), test.contains)) {
				t.Fatalf("check error %q does not contain %q", err, test.contains)
			}
		})
	}
}

func TestPackageOrganizationMatchesAgentProviderLayout(t *testing.T) {
	for _, test := range []struct {
		name     string
		files    func() map[string]string
		invalid  bool
		contains string
	}{
		{
			name:  "current agent provider layout",
			files: agentProviderLayoutFiles,
		},
		{
			name: "provider options remain at llm root",
			files: func() map[string]string {
				files := agentProviderLayoutFiles()
				delete(files, "internal/agents/llm/option.go")
				return files
			},
			invalid:  true,
			contains: "internal/agents/llm must contain Go source",
		},
		{
			name: "Codex implements runner contract",
			files: func() map[string]string {
				files := agentProviderLayoutFiles()
				files["internal/agents/llm/codex/codex_cli_runner.go"] = "package codex\n\ntype CodexCLIRunner struct{}\n"
				return files
			},
			invalid:  true,
			contains: "internal/agents/llm/codex must declare a struct that asserts agents.AgentRunner",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := checkPackageOrganization(fixtureRepository(t, test.files()))
			if (err != nil) != test.invalid {
				t.Fatalf("check returned %v", err)
			}
			if test.contains != "" && (err == nil || !strings.Contains(err.Error(), test.contains)) {
				t.Fatalf("check error %q does not contain %q", err, test.contains)
			}
		})
	}
}

func TestPackageOrganizationRespectsConfiguredRoots(t *testing.T) {
	original := packageOwnership
	defer func() { packageOwnership = original }()
	packageOwnership = packageOwnershipSettings{
		Agents:    "contracts",
		Options:   "providers",
		SharedCLI: "providers/runnercli",
		Option:    "ProviderOption",
		Runner:    "Runner",
		Providers: []string{"providers/alpha", "providers/beta"},
		Retired:   []string{"legacy"},
		Contracts: []string{"Agent", "Runner"},
	}
	files := map[string]string{
		"contracts/agent.go":         "package contracts\n\ntype Agent interface{}\n",
		"contracts/runner.go":        "package contracts\n\ntype Runner interface{}\n",
		"contracts/rule_agent.go":    "package contracts\n\ntype RuleAgent struct{}\n",
		"providers/option.go":        "package providers\n\ntype ProviderOption func()\n",
		"providers/runnercli/cli.go": "package runnercli\n\ntype CLI struct{}\n",
		"providers/alpha/runner.go":  providerFixture("alpha", "AlphaRunner"),
		"providers/beta/runner.go":   providerFixture("beta", "BetaRunner"),
	}
	if err := checkPackageOrganization(fixtureRepository(t, files)); err != nil {
		t.Fatalf("configured package layout returned %v", err)
	}
	violating := mergeFixtureFiles(files, map[string]string{
		"legacy/agent.go": "package legacy\n",
	})
	if err := checkPackageOrganization(fixtureRepository(t, violating)); err == nil || !strings.Contains(err.Error(), "retired package") {
		t.Fatalf("retired configured package returned %v", err)
	}
}

func TestBootstrapOwnsConcreteProviderImports(t *testing.T) {
	allowed := map[string]string{
		"bootstrap.go": `package main

import (
	_ "example.com/fixture/internal/agents/llm/claude"
	_ "example.com/fixture/internal/agents/llm/codex"
)

func bootstrap() {}
`,
		"internal/agents/llm/claude/claude_cli_runner.go": "package claude\n",
		"internal/agents/llm/codex/codex_cli_runner.go": `package codex

import _ "example.com/fixture/internal/agents/llm/runnercli"
`,
		"internal/agents/llm/runnercli/cli.go": "package runnercli\n",
	}
	if err := checkBootstrap(fixtureRepository(t, allowed)); err != nil {
		t.Fatalf("bootstrap provider selection or provider helper import returned %v", err)
	}
	violating := mergeFixtureFiles(allowed, map[string]string{
		"internal/commands/root.go": "package commands\n\nimport _ \"example.com/fixture/internal/agents/llm/codex\"\n",
	})
	if err := checkBootstrap(fixtureRepository(t, violating)); err == nil || !strings.Contains(err.Error(), "concrete implementation") {
		t.Fatalf("non-bootstrap provider import returned %v", err)
	}
}

func agentProviderLayoutFiles() map[string]string {
	return map[string]string{
		"internal/agents/agent.go":             "package agents\n\ntype Agent interface{}\n",
		"internal/agents/agent_runner.go":      "package agents\n\ntype AgentRunner interface{}\n",
		"internal/agents/rule_agent.go":        "package agents\n\ntype RuleAgent struct{}\n",
		"internal/agents/llm/option.go":        "package llm\n\ntype Option func()\n",
		"internal/agents/llm/runnercli/cli.go": "package runnercli\n\ntype CLI struct{}\n",
		"internal/agents/llm/codex/codex_cli_runner.go": `package codex

import (
	"example.com/fixture/internal/agents"
	"example.com/fixture/internal/agents/llm/runnercli"
)

type CodexCLIRunner struct { cli *runnercli.CLI }

var _ agents.AgentRunner = (*CodexCLIRunner)(nil)
`,
		"internal/agents/llm/claude/claude_cli_runner.go": `package claude

import (
	"example.com/fixture/internal/agents"
	"example.com/fixture/internal/agents/llm/runnercli"
)

type ClaudeCLIRunner struct { cli *runnercli.CLI }

var _ agents.AgentRunner = (*ClaudeCLIRunner)(nil)
`,
	}
}

func providerFixture(packageName, runner string) string {
	return "package " + packageName + "\n\nimport \"example.com/fixture/contracts\"\n\ntype " + runner + " struct{}\n\nvar _ contracts.Runner = (*" + runner + ")(nil)\n"
}
