package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	dir := t.TempDir()
	write := func(name, source string) string {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
			t.Fatal(err)
		}
		return path
	}
	changed := write("changed.liz", "let x = 10\n-1\nlet t = a\n  + b\n")
	same := write("same.liz", "let x = 10 -\n  1\nif (x) { 1 }\n-1\n")

	tests := []struct {
		name     string
		paths    []string
		status   int
		expected []string
	}{
		{"nothing to change", []string{same}, 0, nil},
		{"changed lines", []string{changed, same}, 1, []string{
			changed + ": [line 2, column 1] this line no longer goes on with the line before: - starts a new statement; " +
				"to go on, end the line before with an operator or join the lines",
			changed + ": [line 4, column 3] a line can't start with +; put the + at the end of the previous line",
			"2 lines to change in 1 files",
		}},
		{"missing file", []string{filepath.Join(dir, "missing.liz")}, 2, nil},
	}
	for _, tt := range tests {
		var out, errOut bytes.Buffer
		status := run(tt.paths, &out, &errOut)
		if status != tt.status {
			t.Errorf("%s: status %d, want %d", tt.name, status, tt.status)
		}
		want := strings.Join(tt.expected, "\n")
		if want != "" {
			want += "\n"
		}
		if out.String() != want {
			t.Errorf("%s:\nexpected:\n%s\n     got:\n%s", tt.name, want, out.String())
		}
	}
}
