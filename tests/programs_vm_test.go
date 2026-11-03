package tests

import (
	"bytes"
	"lizardscript/repl"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The same programs on the bytecode VM: they must print exactly what the
// .expected files say, like on the evaluator.

func TestValidProgramsVM(t *testing.T) {
	runProgramsVM(t, "valid", true)
}

func TestErrorProgramsVM(t *testing.T) {
	runProgramsVM(t, "errors", false)
}

func runProgramsVM(t *testing.T, dir string, shouldSucceed bool) {
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

			var output bytes.Buffer
			succeeded := repl.RunFile(filepath.ToSlash(file), normalize(string(source)), &output, &output, true)
			if succeeded != shouldSucceed {
				t.Errorf("expected success to be %t, got %t. Output:\n%s", shouldSucceed, succeeded, output.String())
			}

			expected, err := os.ReadFile(strings.TrimSuffix(file, ".liz") + ".expected")
			if err != nil {
				t.Fatal(err)
			}
			if got, want := output.String(), normalize(string(expected)); got != want {
				t.Errorf("wrong output\n--- expected:\n%s\n--- got:\n%s", want, got)
			}
		})
	}
}
