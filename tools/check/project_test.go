package check

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestProjectNames checks that a file finds the nearest project file, in
// its folder or above, and that File knows its names.
func TestProjectNames(t *testing.T) {
	dir := t.TempDir()
	write := func(name, source string) string {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
			t.Fatal(err)
		}
		return path
	}
	write("app/lizard-check.liz", "-~> The names of the host.\n{\"names\": [\"canvas\", \"settings\"]}\n")
	write("app/inner/lizard-check.liz", "{\"names\": [\"inner\"]}")
	app := write("app/ui/view.liz", "canvas.fill(settings)\nsettings = 2\n")
	inner := write("app/inner/deep/a.liz", "puts(inner, canvas)\n")
	alone := write("alone.liz", "puts(canvas)\n")

	tests := []struct {
		name, path string
		expected   []string
	}{
		{"names from a folder above", app, nil},
		{"the nearest project file only", inner, []string{"1:13 error unknown-name: identifier not found: canvas"}},
		{"no project file", alone, []string{"1:6 error unknown-name: identifier not found: canvas"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			found, err := File(tt.path)
			if err != nil {
				t.Fatal(err)
			}
			var got []string
			for _, d := range found {
				got = append(got, show(d))
			}
			if strings.Join(got, "\n") != strings.Join(tt.expected, "\n") {
				t.Errorf("got\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(tt.expected, "\n"))
			}
		})
	}
}

func TestProjectFileMistakes(t *testing.T) {
	tests := []struct{ source, expected string }{
		{`{"names": ["a", "b"]}`, ""},
		{"-~> nothing\n{}", ""},
		{`["a"]`, `it must be one hash, like {"names": ["canvas"]}`},
		{`{"names": ["a"]}` + "\n1", `it must be one hash, like {"names": ["canvas"]}`},
		{`{"name": ["a"]}`, `[line 1, column 2] the only key is "names", got "name"`},
		{`{"names": "a"}`, `[line 1, column 11] names must be a list of strings, got "a"`},
		{`{"names": ["a", 1]}`, `[line 1, column 17] names must be a list of strings, got 1 in it`},
		{`{"names": [`, "[line 1, column 12] syntax error"},
	}
	for _, tt := range tests {
		_, err := parseProject(tt.source)
		got := ""
		if err != nil {
			got = err.Error()
		}
		if !strings.HasPrefix(got, tt.expected) || (tt.expected == "") != (got == "") {
			t.Errorf("%q: got %q, want %q", tt.source, got, tt.expected)
		}
	}
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, ProjectFile), []byte("[1]"), 0o644)
	os.WriteFile(filepath.Join(dir, "a.liz"), []byte("1"), 0o644)
	if _, err := File(filepath.Join(dir, "a.liz")); err == nil || !strings.Contains(err.Error(), "lizard-check.liz: it must be one hash") {
		t.Errorf("a broken project file gives %v", err)
	}
}
