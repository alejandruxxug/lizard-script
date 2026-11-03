package main

import (
	"os"
	"testing"
)

// TestGeneratedFileIsCurrent fails when docs/SEMANTICS.md or
// docs/TERRARIUM.md changed and lsp/builtin_docs.go was not made again
// (go generate ./lsp).
func TestGeneratedFileIsCurrent(t *testing.T) {
	want, err := generate("../../../docs")
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile("../../builtin_docs.go")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Fatal("lsp/builtin_docs.go is out of date: run go generate ./lsp")
	}
}

func TestReadsBothDocuments(t *testing.T) {
	entries := map[string]*entry{}
	readTables("## The math library\n\n| Name | Does |\n| --- | --- |\n| `math.sin(x)`, `math.cos(x)` | As a float. |\n| `math.pi` | π. |\n\n## Other\n\n| `skip(x)` | no |\n", entries)
	modules := readHeadings("| [`fs`](#fs) | files and folders |\n\n## fs\n\n### fs.read(path)\nThe text, see [errors](#errors).\n\n### clip(target, rect)\n### unclip(target)\nCuts.\n", entries)

	tests := []struct {
		name       string
		signatures int
		doc        string
	}{
		{"math.sin", 1, "As a float."},
		{"math.cos", 1, "As a float."},
		{"math.pi", 0, "π."},
		{"fs.read", 1, "The text, see errors."},
		{"*.clip", 1, "Cuts."},
		{"*.unclip", 1, "Cuts."},
	}
	for _, tt := range tests {
		e := entries[tt.name]
		if e == nil {
			t.Errorf("%s: missing", tt.name)
			continue
		}
		if len(e.signatures) != tt.signatures || e.doc != tt.doc {
			t.Errorf("%s: got %v %q, want %d signatures and %q", tt.name, e.signatures, e.doc, tt.signatures, tt.doc)
		}
	}
	if entries["skip"] != nil {
		t.Error("a table outside the builtin sections was read")
	}
	if modules["fs"] != "files and folders" {
		t.Errorf("module summary: got %q", modules["fs"])
	}
}
