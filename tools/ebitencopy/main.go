// Command ebitencopy makes third_party/ebiten, the copy of Ebitengine that
// go.mod's replace points to: the version go.mod requires, with only the
// packages lizard-script needs (on any system, as go mod tidy counts them)
// and one change, a Metal device created when the first window opens
// instead of when the program starts (see third_party/README.md).
//
//	go run ./tools/ebitencopy
//
// It runs from the repository's folder, after the version in go.mod's
// require line changes; then go mod tidy and the tests.
package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// module is Ebitengine's module path, and target the copy's folder.
const (
	module = "github.com/hajimehoshi/ebiten/v2"
	target = "third_party/ebiten"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "ebitencopy:", err)
		os.Exit(1)
	}
}

func run() error {
	version, err := requiredVersion("go.mod")
	if err != nil {
		return err
	}
	source, err := download(version)
	if err != nil {
		return err
	}
	direct, err := importedBy(".")
	if err != nil {
		return err
	}
	packages, err := closure(source, direct)
	if err != nil {
		return err
	}
	if err := os.RemoveAll(target); err != nil {
		return err
	}
	for _, pkg := range packages {
		if err := copyPackage(source, pkg); err != nil {
			return err
		}
	}
	if err := patchFile(filepath.Join(target, patch.file)); err != nil {
		return err
	}
	fmt.Printf("copied %d packages of %s %s to %s\n", len(packages), module, version, target)
	return nil
}

// requiredVersion is the version of Ebitengine that go.mod requires.
func requiredVersion(gomod string) (string, error) {
	f, err := os.Open(gomod)
	if err != nil {
		return "", err
	}
	defer f.Close()
	lines := bufio.NewScanner(f)
	for lines.Scan() {
		fields := strings.Fields(strings.TrimPrefix(strings.TrimSpace(lines.Text()), "require"))
		if len(fields) >= 2 && fields[0] == module {
			return fields[1], nil
		}
	}
	return "", fmt.Errorf("go.mod doesn't require %s", module)
}

// download is the folder of the module at version in Go's module cache.
func download(version string) (string, error) {
	out, err := exec.Command("go", "mod", "download", "-json", module+"@"+version).Output()
	if err != nil {
		return "", fmt.Errorf("go mod download: %w", err)
	}
	var info struct{ Dir, Error string }
	if err := json.Unmarshal(out, &info); err != nil {
		return "", err
	}
	if info.Error != "" {
		return "", errors.New(info.Error)
	}
	return info.Dir, nil
}

// importedBy are the packages of Ebitengine that the Go files under dir
// import, leaving the copy out.
func importedBy(dir string) ([]string, error) {
	seen := map[string]bool{}
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && (filepath.ToSlash(path) == "third_party" || strings.HasPrefix(d.Name(), ".")) && path != "." {
			return filepath.SkipDir
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		imports, err := importsOf(path)
		for _, imp := range imports {
			seen[imp] = true
		}
		return err
	})
	return sorted(seen), err
}

// closure is the packages of the module at source that the packages in
// start need, start included, reading every Go file but the tests,
// whatever its build constraints.
func closure(source string, start []string) ([]string, error) {
	seen := map[string]bool{}
	for todo := start; len(todo) > 0; {
		pkg := todo[0]
		todo = todo[1:]
		if seen[pkg] {
			continue
		}
		seen[pkg] = true
		files, err := filepath.Glob(filepath.Join(source, relative(pkg), "*.go"))
		if err != nil {
			return nil, err
		}
		for _, file := range files {
			if strings.HasSuffix(file, "_test.go") {
				continue
			}
			imports, err := importsOf(file)
			if err != nil {
				return nil, err
			}
			todo = append(todo, imports...)
		}
	}
	return sorted(seen), nil
}

// importsOf are the packages of Ebitengine that a Go file imports.
func importsOf(file string) ([]string, error) {
	parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.ImportsOnly)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, imp := range parsed.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		if path == module || strings.HasPrefix(path, module+"/") {
			out = append(out, path)
		}
	}
	return out, nil
}

// copyPackage copies the files of a package's folder, but its tests and
// hidden files, to the copy. The module's own folder brings go.mod, go.sum,
// LICENSE and NOTICE.md with it.
func copyPackage(source, pkg string) error {
	from, to := filepath.Join(source, relative(pkg)), filepath.Join(target, relative(pkg))
	entries, err := os.ReadDir(from)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(to, 0o755); err != nil {
		return err
	}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || strings.HasSuffix(name, "_test.go") || strings.HasPrefix(name, ".") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(from, name))
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(to, name), data, 0o644); err != nil {
			return err
		}
	}
	return nil
}

// relative is the folder of a package inside the module.
func relative(pkg string) string {
	return filepath.FromSlash(strings.TrimPrefix(strings.TrimPrefix(pkg, module), "/"))
}

func sorted(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for s := range set {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}
