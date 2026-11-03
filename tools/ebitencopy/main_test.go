package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApply(t *testing.T) {
	tests := []struct {
		name, text string
		edits      []edit
		want, err  string
	}{
		{"one edit", "a b c", []edit{{"b", "x"}}, "a x c", ""},
		{"in order", "a b c", []edit{{"b", "x"}, {"x c", "y"}}, "a y", ""},
		{"missing", "a b c", []edit{{"z", "x"}}, "", "edit 1 of the patch is found 0 times"},
		{"twice", "b b", []edit{{"b", "x"}}, "", "edit 1 of the patch is found 2 times"},
	}
	for _, tt := range tests {
		got, err := apply(tt.text, tt.edits)
		if tt.err != "" {
			if err == nil || !strings.HasPrefix(err.Error(), tt.err) {
				t.Errorf("%s: got error %v, want %q", tt.name, err, tt.err)
			}
			continue
		}
		if err != nil || got != tt.want {
			t.Errorf("%s: got %q, %v, want %q", tt.name, got, err, tt.want)
		}
	}
}

// TestTheCopyIsPatched checks that the copy go.mod points to has the
// patch, and the version go.mod requires.
func TestTheCopyIsPatched(t *testing.T) {
	root := filepath.Join("..", "..")
	data, err := os.ReadFile(filepath.Join(root, target, patch.file))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range patch.edits {
		if !strings.Contains(string(data), e.new) {
			t.Errorf("the copy doesn't have the change %q", strings.SplitN(e.new, "\n", 2)[0])
		}
	}
	gomod, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(gomod), "replace "+module+" => ./"+target) {
		t.Errorf("go.mod doesn't replace %s with ./%s", module, target)
	}
}
