package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

type repository struct {
	root     string
	module   string
	dataDir  string
	rulesDir string
	files    []*sourceFile
	types    map[string]map[string]typeDefinition
	loaded   bool
	loadErr  error
	tested   bool
	testErr  error
}

type sourceFile struct {
	name    string
	dir     string
	tree    *ast.File
	tokens  *token.FileSet
	imports map[string]string
	lines   int
	test    bool
}

type typeDefinition struct {
	source *sourceFile
	spec   *ast.TypeSpec
}

func newRepository(root, dataDir, rulesDir string) (*repository, error) {
	if !filepath.IsLocal(dataDir) || !filepath.IsLocal(rulesDir) || rulesDir == "." {
		return nil, fmt.Errorf("data-dir and rules-dir must be project-relative")
	}
	repo := &repository{root: root, dataDir: filepath.ToSlash(dataDir), rulesDir: rulesDir}
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err == nil {
		cmd := exec.Command("go", "list", "-m", "-f", "{{.Path}}")
		cmd.Dir = root
		output, err := cmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("read Go module: %w\n%s", err, output)
		}
		repo.module = strings.TrimSpace(string(output))
	}
	return repo, nil
}

func within(name, root string) bool {
	return root == "." || name == root || strings.HasPrefix(name, root+"/")
}

func (repo *repository) sources(includeTests bool) ([]*sourceFile, error) {
	if !repo.loaded {
		repo.loaded = true
		repo.types = map[string]map[string]typeDefinition{}
		repo.loadErr = filepath.WalkDir(repo.root, repo.collectSource)
	}
	if repo.loadErr != nil {
		return nil, repo.loadErr
	}
	files := make([]*sourceFile, 0, len(repo.files))
	for _, file := range repo.files {
		if includeTests || !file.test {
			files = append(files, file)
		}
	}
	return files, nil
}

func (repo *repository) collectSource(name string, entry fs.DirEntry, err error) error {
	if err != nil {
		return err
	}
	relative, err := filepath.Rel(repo.root, name)
	if err != nil {
		return err
	}
	relative = filepath.ToSlash(relative)
	if entry.IsDir() {
		generated := within(relative, repo.rulesDir) ||
			within(relative, path.Join(repo.dataDir, ".tmp")) ||
			within(relative, path.Join(repo.dataDir, "bin"))
		artifactRoot := repo.dataDir != "." && within(relative, repo.dataDir)
		if relative != "." && (strings.HasPrefix(entry.Name(), ".") || entry.Name() == "vendor" || artifactRoot || generated) {
			return filepath.SkipDir
		}
		return nil
	}
	if !strings.HasSuffix(entry.Name(), ".go") {
		return nil
	}
	data, err := os.ReadFile(name)
	if err != nil {
		return err
	}
	tokens := token.NewFileSet()
	tree, err := parser.ParseFile(tokens, name, data, parser.ParseComments)
	if err != nil {
		return err
	}
	if ast.IsGenerated(tree) {
		return nil
	}
	file := &sourceFile{
		name: relative, dir: path.Dir(relative), tree: tree, tokens: tokens,
		imports: map[string]string{}, lines: bytes.Count(data, []byte{'\n'}), test: strings.HasSuffix(name, "_test.go"),
	}
	if len(data) > 0 && data[len(data)-1] != '\n' {
		file.lines++
	}
	for _, spec := range tree.Imports {
		imported := strings.Trim(spec.Path.Value, "\"")
		alias := path.Base(imported)
		if spec.Name != nil {
			alias = spec.Name.Name
		}
		file.imports[alias] = imported
	}
	repo.files = append(repo.files, file)
	if !file.test {
		repo.indexTypes(file)
	}
	return nil
}

func (repo *repository) indexTypes(file *sourceFile) {
	if repo.types[file.dir] == nil {
		repo.types[file.dir] = map[string]typeDefinition{}
	}
	for _, declaration := range file.tree.Decls {
		gen, ok := declaration.(*ast.GenDecl)
		if !ok || gen.Tok != token.TYPE {
			continue
		}
		for _, specification := range gen.Specs {
			spec := specification.(*ast.TypeSpec)
			repo.types[file.dir][spec.Name.Name] = typeDefinition{file, spec}
		}
	}
}

func (repo *repository) packagePath(dir string) string {
	if dir == "." {
		return repo.module
	}
	if repo.module == "" {
		return dir
	}
	return repo.module + "/" + dir
}

func (repo *repository) compile() error {
	if repo.tested {
		return repo.testErr
	}
	repo.tested = true
	cmd := exec.Command("go", "test", "./...")
	cmd.Dir = repo.root
	output, err := cmd.CombinedOutput()
	if err != nil {
		repo.testErr = fmt.Errorf("compiler conformance check failed: %w\n%s", err, output)
	}
	return repo.testErr
}

func (file *sourceFile) at(position token.Pos, message string) error {
	line := file.tokens.PositionFor(position, false).Line
	return fmt.Errorf("%s:%d: %s", file.name, line, message)
}
