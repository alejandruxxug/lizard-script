// Package build is lizardscript build: it makes one executable file of a
// program folder, the program's main.liz, every module it uses (and the
// ones they use) and the data files its bundle.txt lists, for this system
// or another one. The file is a copy of lizardscript with the program
// appended (package bundle), so it runs with nothing else installed.
//
// With an icon (Options.Icon) the program also gets its icon on each system,
// and on macOS the output is an app folder; see icon.go and app.go.
//
// bundle.txt has one path or pattern per line, relative to the folder,
// like assets/*.png; a folder takes everything in it, and a line that
// starts with -~> is a comment.
package build

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"github.com/alejandruxxug/lizard-script/engine/evaluator"
	"github.com/alejandruxxug/lizard-script/lang/ast"
	"github.com/alejandruxxug/lizard-script/lang/lexer"
	"github.com/alejandruxxug/lizard-script/lang/parser"
	"github.com/alejandruxxug/lizard-script/tools/bundle"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// Options say what to build.
type Options struct {
	Dir    string // the program's folder, with its main.liz
	Output string // the file to make
	GOOS   string // the system to build for, like "windows"; "" is this one
	GOARCH string // the processor, like "amd64"; "" is this one
	VM     bool   // the program runs on the bytecode VM
	// Runtime is a lizardscript built for the target system, to copy. When
	// it is "" the running lizardscript is copied for this system, and for
	// another one lizardscript is built from its source with Go.
	Runtime string
	// Icon is a square PNG of at least 256 pixels (1024 is best), the
	// program's icon; IconSmall is an optional hand-drawn one for 16 pixels.
	// On macOS the output must be called Name.app: Build makes an app
	// folder, with or without an icon.
	Icon      string
	IconSmall string
}

// Build makes the executable of o and returns the files it bundled.
func Build(o Options) ([]string, error) {
	if o.GOOS == "" {
		o.GOOS = runtime.GOOS
	}
	if o.GOARCH == "" {
		o.GOARCH = runtime.GOARCH
	}
	files, err := Collect(o.Dir)
	if err != nil {
		return nil, err
	}
	base, err := runtimeFor(o)
	if err != nil {
		return nil, err
	}
	var icons *iconSet
	if o.Icon != "" {
		if icons, err = loadIcons(o.Icon, o.IconSmall); err != nil {
			return nil, err
		}
	} else if o.IconSmall != "" {
		return nil, errors.New("--icon-small needs --icon")
	}
	if err := write(o, base, bundle.Program{Main: "main.liz", VM: o.VM}, files, icons); err != nil {
		return nil, err
	}
	names := make([]string, len(files))
	for i, f := range files {
		names[i] = f.Name
	}
	return names, nil
}

// write makes the output of o for its system: an app folder on macOS, else
// one file, with the icon added to the file on Windows or beside it on Linux.
func write(o Options, base []byte, program bundle.Program, files []bundle.File, icons *iconSet) error {
	var archive bytes.Buffer
	if err := bundle.Write(&archive, program, files); err != nil {
		return err
	}
	app := strings.HasSuffix(o.Output, appSuffix)
	switch {
	case o.GOOS == "darwin" && app:
		return writeApp(o, base, archive.Bytes(), icons)
	case o.GOOS == "darwin" && icons != nil:
		return fmt.Errorf("an icon on macOS needs an app: make the output %s.app", strings.TrimSuffix(filepath.Base(o.Output), filepath.Ext(o.Output)))
	case app:
		return fmt.Errorf("%s is an app, which only a build for macOS makes (--os darwin)", o.Output)
	}
	if o.GOOS == "windows" && icons != nil {
		var err error
		if base, err = addIcon(base, icoImages(icons)); err != nil {
			return fmt.Errorf("adding the icon: %w", err)
		}
	}
	out := append(append([]byte{}, base...), archive.Bytes()...)
	if err := os.WriteFile(o.Output, out, 0o755); err != nil {
		return err
	}
	if o.GOOS == "linux" && icons != nil {
		return writeLinuxIcon(o.Output, icons)
	}
	return nil
}

// Collect finds the files of the program in dir: main.liz, the modules it
// uses, and the data files of bundle.txt, in order of their names.
func Collect(dir string) ([]bundle.File, error) {
	root, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	found := map[string][]byte{}
	if err := collectModules(root, found); err != nil {
		return nil, err
	}
	if err := collectData(root, found); err != nil {
		return nil, err
	}
	names := make([]string, 0, len(found))
	for name := range found {
		names = append(names, name)
	}
	sort.Strings(names)
	files := make([]bundle.File, len(names))
	for i, name := range names {
		files[i] = bundle.File{Name: name, Data: found[name]}
	}
	return files, nil
}

// collectModules adds main.liz and every module it uses, following each
// use the way the program will at run time.
func collectModules(root string, found map[string][]byte) error {
	main := filepath.Join(root, "main.liz")
	source, err := os.ReadFile(main)
	if err != nil {
		return fmt.Errorf("%s has no main.liz: %w", root, err)
	}
	queue := []string{main}
	sources := map[string][]byte{main: source}
	for len(queue) > 0 {
		file := queue[0]
		queue = queue[1:]
		name, err := inside(root, file)
		if err != nil {
			return err
		}
		if _, done := found[name]; done {
			continue
		}
		found[name] = sources[file]
		uses, err := usesOf(file, sources[file])
		if err != nil {
			return err
		}
		for _, use := range uses {
			path, module, ferr := evaluator.FindModule(use)
			if ferr != nil {
				return fmt.Errorf("%s: %s", relative(root, file), ferr.Message)
			}
			if evaluator.IsStandardModule(path) {
				// Every lizardscript has the standard modules.
				continue
			}
			path = filepath.FromSlash(path)
			if _, err := inside(root, path); err != nil {
				return fmt.Errorf("%s uses %s, which is outside the program's folder", relative(root, file), relative(root, path))
			}
			sources[path] = module
			queue = append(queue, path)
		}
	}
	return nil
}

// usesOf parses a file and returns its use statements.
func usesOf(file string, source []byte) ([]*ast.UseStatement, error) {
	p := parser.New(lexer.NewNamed(string(source), filepath.ToSlash(file)))
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		return nil, fmt.Errorf("%s: %s", file, strings.Join(p.Errors(), "\n"))
	}
	var uses []*ast.UseStatement
	for _, statement := range program.Statements {
		if use, ok := statement.(*ast.UseStatement); ok {
			uses = append(uses, use)
		}
	}
	return uses, nil
}

// collectData adds the files that bundle.txt lists.
func collectData(root string, found map[string][]byte) error {
	list, err := os.ReadFile(filepath.Join(root, "bundle.txt"))
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	lines := bufio.NewScanner(bytes.NewReader(list))
	for n := 1; lines.Scan(); n++ {
		line := strings.TrimSpace(lines.Text())
		if line == "" || strings.HasPrefix(line, "-~>") {
			continue
		}
		matches, err := filepath.Glob(filepath.Join(root, filepath.FromSlash(line)))
		if err != nil || len(matches) == 0 {
			return fmt.Errorf("bundle.txt, line %d: nothing is called %s", n, line)
		}
		for _, match := range matches {
			if err := addTree(root, match, found); err != nil {
				return fmt.Errorf("bundle.txt, line %d: %w", n, err)
			}
		}
	}
	return lines.Err()
}

// addTree adds a file, or every file in a folder.
func addTree(root, path string, found map[string][]byte) error {
	return filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		name, err := inside(root, p)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		found[name] = data
		return nil
	})
}

// inside is the name of path in the program's folder, with "/", or an error
// when it is outside of it.
func inside(root, path string) (string, error) {
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%s is outside the program's folder %s", path, root)
	}
	return filepath.ToSlash(rel), nil
}

// relative is path from root, for messages.
func relative(root, path string) string {
	if rel, err := filepath.Rel(root, path); err == nil {
		return filepath.ToSlash(rel)
	}
	return path
}

// runtimeFor is the lizardscript to copy for the target of o, without any
// program a built one carries.
func runtimeFor(o Options) ([]byte, error) {
	file := o.Runtime
	if file == "" && o.GOOS == runtime.GOOS && o.GOARCH == runtime.GOARCH {
		self, err := os.Executable()
		if err != nil {
			return nil, err
		}
		file = self
	}
	if file == "" {
		built, err := buildRuntime(o.GOOS, o.GOARCH)
		if err != nil {
			return nil, err
		}
		defer os.RemoveAll(filepath.Dir(built))
		file = built
	}
	size, err := bundle.RuntimeSize(file)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(io.LimitReader(f, size))
}

// buildRuntime builds lizardscript for another system from its source,
// found by Source, with Go, and returns the file, in a new temporary folder.
func buildRuntime(goos, goarch string) (string, error) {
	src := Source()
	if src == "" {
		return "", fmt.Errorf("to build for %s/%s lizardscript needs its source and Go: run it inside a lizard-script folder or set LIZARD_SRC, or give --runtime with a lizardscript built for %s/%s", goos, goarch, goos, goarch)
	}
	tmp, err := os.MkdirTemp("", "lizardscript-build-")
	if err != nil {
		return "", err
	}
	out := filepath.Join(tmp, "lizardscript")
	cmd := exec.Command("go", "build", "-trimpath", "-o", out, ".")
	cmd.Dir = src
	cmd.Env = append(os.Environ(), "GOOS="+goos, "GOARCH="+goarch, "CGO_ENABLED=0")
	if text, err := cmd.CombinedOutput(); err != nil {
		os.RemoveAll(tmp)
		return "", fmt.Errorf("building lizardscript for %s/%s: %v\n%s", goos, goarch, err, text)
	}
	return out, nil
}

// Source is the folder of lizard-script's source: $LIZARD_SRC, or the
// first folder with lizard-script's go.mod in or above the current folder
// or the running program's. It is "" when there is none.
func Source() string {
	if dir := os.Getenv("LIZARD_SRC"); dir != "" && isSource(dir) {
		return dir
	}
	var starts []string
	if wd, err := os.Getwd(); err == nil {
		starts = append(starts, wd)
	}
	if self, err := os.Executable(); err == nil {
		starts = append(starts, filepath.Dir(self))
	}
	for _, start := range starts {
		for dir := start; ; dir = filepath.Dir(dir) {
			if isSource(dir) {
				return dir
			}
			if filepath.Dir(dir) == dir {
				break
			}
		}
	}
	return ""
}

// isSource reports whether dir holds lizard-script's go.mod.
func isSource(dir string) bool {
	mod, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		return false
	}
	first, _, _ := strings.Cut(string(mod), "\n")
	return strings.Join(strings.Fields(first), " ") == "module github.com/alejandruxxug/lizard-script"
}
