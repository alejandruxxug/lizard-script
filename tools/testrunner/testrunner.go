// Package testrunner is lizardscript test: it runs the test programs of a
// folder and compares what they print with what they should.
//
// A test is a program.liz with a program.expected next to it, holding its
// exact output: what it prints, followed by its error messages, if any
// (like the programs of tests/ in this repository). A program.liz without
// one is not a test, so the modules a test uses can sit next to it. A
// program_bench.liz is a benchmark: with Bench set it runs, and its time is
// shown; it doesn't need an .expected file.
package testrunner

import (
	"bytes"
	"fmt"
	"github.com/alejandruxxug/lizard-script/terrarium"
	"github.com/alejandruxxug/lizard-script/tools/repl"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Options say what to run and how.
type Options struct {
	Dir    string    // the folder, searched with its subfolders
	VM     bool      // run on the bytecode VM
	Bench  bool      // run the benchmarks too
	Update bool      // rewrite each .expected file with the output
	Out    io.Writer // where the report goes
}

// Run runs the tests and benchmarks of o.Dir and reports each one. It
// returns how many failed; err is for a folder that can't be read.
func Run(o Options) (failed int, err error) {
	tests, benches, err := find(o.Dir)
	if err != nil {
		return 0, err
	}
	if !o.Bench {
		benches = nil
	}
	if len(tests) == 0 && len(benches) == 0 {
		fmt.Fprintf(o.Out, "no tests in %s\n", o.Dir)
		return 0, nil
	}
	passed := 0
	for _, file := range tests {
		if runTest(o, file) {
			passed++
		}
	}
	for _, file := range benches {
		if runBench(o, file) {
			passed++
		}
	}
	failed = len(tests) + len(benches) - passed
	fmt.Fprintf(o.Out, "%d passed, %d failed\n", passed, failed)
	return failed, nil
}

// find lists the tests and the benchmarks under dir, in order. Folders
// whose name starts with "." are skipped.
func find(dir string) (tests, benches []string, err error) {
	err = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != dir && strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		switch {
		case strings.HasSuffix(path, "_bench.liz"):
			benches = append(benches, path)
		case strings.HasSuffix(path, ".liz"):
			if _, err := os.Stat(expectedFile(path)); err == nil {
				tests = append(tests, path)
			}
		}
		return nil
	})
	sort.Strings(tests)
	sort.Strings(benches)
	return tests, benches, err
}

func expectedFile(program string) string {
	return strings.TrimSuffix(program, ".liz") + ".expected"
}

// runTest runs one test and reports it.
func runTest(o Options, file string) bool {
	name := filepath.ToSlash(file)
	output, _, elapsed, err := run(o, file)
	if err != nil {
		fmt.Fprintf(o.Out, "FAIL  %s: %v\n", name, err)
		return false
	}
	if o.Update {
		if err := os.WriteFile(expectedFile(file), []byte(output), 0o644); err != nil {
			fmt.Fprintf(o.Out, "FAIL  %s: %v\n", name, err)
			return false
		}
	}
	expected, err := os.ReadFile(expectedFile(file))
	if err != nil {
		fmt.Fprintf(o.Out, "FAIL  %s: %v\n", name, err)
		return false
	}
	want := normalize(string(expected))
	if output != want {
		fmt.Fprintf(o.Out, "FAIL  %s\n--- expected:\n%s--- got:\n%s", name, endLine(want), endLine(output))
		return false
	}
	fmt.Fprintf(o.Out, "ok    %s (%s)\n", name, milliseconds(elapsed))
	return true
}

// runBench runs one benchmark and shows its time. It fails when the
// program does.
func runBench(o Options, file string) bool {
	name := filepath.ToSlash(file)
	output, ok, elapsed, err := run(o, file)
	if err != nil {
		fmt.Fprintf(o.Out, "FAIL  %s: %v\n", name, err)
		return false
	}
	if !ok {
		fmt.Fprintf(o.Out, "FAIL  %s\n%s", name, endLine(output))
		return false
	}
	fmt.Fprintf(o.Out, "bench %s %s\n", name, milliseconds(elapsed))
	return true
}

// exit is what os.exit does inside a test: it stops the program, not the
// runner.
type exit struct{ code int }

// run runs a program and returns everything it printed, errors included,
// and whether it ended without an error (os.exit(0) counts as one).
func run(o Options, file string) (output string, ok bool, elapsed time.Duration, err error) {
	source, err := os.ReadFile(file)
	if err != nil {
		return "", false, 0, err
	}
	var out bytes.Buffer
	savedExit, savedArgs := terrarium.Exit, terrarium.Args
	terrarium.Exit = func(code int) { panic(exit{code}) }
	terrarium.Args = nil
	defer func() {
		terrarium.Exit, terrarium.Args = savedExit, savedArgs
		if r := recover(); r != nil {
			stop, ok := r.(exit)
			if !ok {
				panic(r)
			}
			output, ok = out.String(), stop.code == 0
		}
	}()
	start := time.Now()
	ok = repl.RunFile(filepath.ToSlash(file), normalize(string(source)), &out, &out, o.VM)
	return out.String(), ok, time.Since(start), nil
}

// normalize turns Windows line endings into "\n".
func normalize(text string) string {
	return strings.ReplaceAll(text, "\r\n", "\n")
}

// endLine adds a line break to text that doesn't end with one.
func endLine(text string) string {
	if text != "" && !strings.HasSuffix(text, "\n") {
		return text + "\n"
	}
	return text
}

func milliseconds(d time.Duration) string {
	return fmt.Sprintf("%.1f ms", float64(d.Microseconds())/1000)
}
