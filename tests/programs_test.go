// Package tests runs whole lizard-script programs and compares what they
// print with what they should print.
//
// Every program.liz in valid/ and errors/ has a program.expected next to it
// with its exact output: what puts writes, followed by the error messages, if
// any. The programs in valid/ must run without errors and the ones in errors/
// must fail. They all run with the rest of the tests:
//
//	go test ./...
//
// After a change that is meant to alter some outputs, rewrite the .expected
// files with the update flag, and check the diff before committing it:
//
//	go test ./tests -update
package tests

import (
	"bytes"
	"flag"
	"lizardscript/repl"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "rewrite the .expected files with the current output")

func TestValidPrograms(t *testing.T) {
	runPrograms(t, "valid", true)
}

func TestErrorPrograms(t *testing.T) {
	runPrograms(t, "errors", false)
}

// runPrograms runs every program in dir. shouldSucceed says whether they
// must run without errors.
func runPrograms(t *testing.T, dir string, shouldSucceed bool) {
	files, err := filepath.Glob(filepath.Join(dir, "*.liz"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no programs found in %s: %v", dir, err)
	}

	for _, file := range files {
		name := strings.TrimSuffix(filepath.Base(file), ".liz")
		t.Run(name, func(t *testing.T) {
			source, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}

			// One buffer for both, so the output and the error stay in
			// the order they happened.
			var output bytes.Buffer
			succeeded := repl.Run(normalize(string(source)), &output, &output)

			expectedFile := strings.TrimSuffix(file, ".liz") + ".expected"
			if *update {
				if err := os.WriteFile(expectedFile, output.Bytes(), 0o644); err != nil {
					t.Fatal(err)
				}
			}

			if succeeded != shouldSucceed {
				t.Errorf("expected success to be %t, got %t. Output:\n%s", shouldSucceed, succeeded, output.String())
			}

			expected, err := os.ReadFile(expectedFile)
			if err != nil {
				t.Fatalf("missing %s: %v", expectedFile, err)
			}
			if got, want := output.String(), normalize(string(expected)); got != want {
				t.Errorf("wrong output\n--- expected:\n%s\n--- got:\n%s", want, got)
			}
		})
	}
}

// normalize turns Windows line endings into "\n", so the files give the same
// result whatever line endings git checked them out with.
func normalize(text string) string {
	return strings.ReplaceAll(text, "\r\n", "\n")
}
