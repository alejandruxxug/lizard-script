package build

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"lizardscript/bundle"
)

// writeTree writes files (name to text) under a new folder.
func writeTree(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, text := range files {
		path := filepath.Join(dir, filepath.FromSlash(name))
		os.MkdirAll(filepath.Dir(path), 0o755)
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func names(files []bundle.File) string {
	var out []string
	for _, f := range files {
		out = append(out, f.Name)
	}
	return strings.Join(out, " ")
}

func TestCollect(t *testing.T) {
	dir := writeTree(t, map[string]string{
		"main.liz":           "use \"lib/a\"\nuse \"lib/b\" as b\nputs(1)",
		"lib/a.liz":          "use \"b\"\nuse \"../shared\"",
		"lib/b.liz":          "let b = 1",
		"shared.liz":         "let s = 1",
		"unused.liz":         "let u = 1",
		"assets/one.png":     "png",
		"assets/sub/two.txt": "two",
		"notes/x.md":         "md",
		"notes/y.txt":        "txt",
		"bundle.txt":         "-~> data files\nassets\n\nnotes/*.md\n",
	})
	files, err := Collect(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := "assets/one.png assets/sub/two.txt lib/a.liz lib/b.liz main.liz notes/x.md shared.liz"
	if got := names(files); got != want {
		t.Errorf("got %s\nwant %s", got, want)
	}
}

func TestCollectErrors(t *testing.T) {
	tests := []struct {
		name  string
		files map[string]string
		want  string
	}{
		{"no main", map[string]string{"other.liz": ""}, "has no main.liz"},
		{"a missing module", map[string]string{"main.liz": "use \"nope\""}, `main.liz: [line 1, column 1] runtime error: can't find the module "nope"`},
		{"outside the folder", map[string]string{"main.liz": "use \"../outside\""}, "main.liz uses ../outside.liz, which is outside the program's folder"},
		{"a syntax error", map[string]string{"main.liz": "use \"m\"", "m.liz": "let = 1"}, "syntax error"},
		{"a missing data file", map[string]string{"main.liz": "", "bundle.txt": "\nnope.png"}, "bundle.txt, line 2: nothing is called nope.png"},
	}
	for _, tt := range tests {
		dir := writeTree(t, tt.files)
		if tt.name == "outside the folder" {
			os.WriteFile(filepath.Join(filepath.Dir(dir), "outside.liz"), nil, 0o644)
		}
		if _, err := Collect(dir); err == nil || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("%s: got %v, want %q", tt.name, err, tt.want)
		}
	}
}

// TestBuild builds with a stand-in runtime and checks the program in it.
func TestBuild(t *testing.T) {
	dir := writeTree(t, map[string]string{"main.liz": "use \"m\"\nputs(m)", "m.liz": "let m = 1"})
	runtimeFile := filepath.Join(t.TempDir(), "runtime")
	os.WriteFile(runtimeFile, []byte("RUNTIME"), 0o755)
	out := filepath.Join(t.TempDir(), "app")
	files, err := Build(Options{Dir: dir, Output: out, Runtime: runtimeFile, VM: true, GOOS: "windows", GOARCH: "amd64"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(files, " ") != "m.liz main.liz" {
		t.Errorf("bundled %v", files)
	}
	data, _ := os.ReadFile(out)
	if !strings.HasPrefix(string(data), "RUNTIME") {
		t.Error("the runtime is not at the start")
	}
	program, err := bundle.Open(out)
	if err != nil || program == nil || !program.VM || program.Main != "main.liz" {
		t.Fatalf("Open: %+v, %v", program, err)
	}
	if info, _ := os.Stat(out); runtime.GOOS != "windows" && info.Mode().Perm()&0o100 == 0 {
		t.Error("the program can't be run")
	}

	// Building from a built program copies only its lizardscript.
	again := filepath.Join(t.TempDir(), "again")
	if _, err := Build(Options{Dir: dir, Output: again, Runtime: out}); err != nil {
		t.Fatal(err)
	}
	if size, _ := bundle.RuntimeSize(again); size != int64(len("RUNTIME")) {
		t.Errorf("the second build's runtime is %d bytes", size)
	}
}

func TestSource(t *testing.T) {
	if src := Source(); src == "" || !isSource(src) {
		t.Errorf("Source() = %q inside the repository", src)
	}
	t.Setenv("LIZARD_SRC", t.TempDir())
	if isSource(os.Getenv("LIZARD_SRC")) {
		t.Error("an empty folder is not the source")
	}
}
