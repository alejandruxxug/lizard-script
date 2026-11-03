// Command checknewlines lists the lines of lizard-script programs whose
// meaning changed with LEP 0001, "a statement ends at the end of its line":
// lines that went on with the line before them and now start a statement of
// their own, or are an error. It is for moving programs written before the
// rule; the fix is to put the operator at the end of the line before.
//
//	go run ./tools/checknewlines file.liz...
//
// It prints one line per change, "file: [line L, column C] message", and
// exits with status 1 when it finds any, 2 when a file can't be read.
package main

import (
	"fmt"
	"io"
	"lizardscript/lexer"
	"lizardscript/parser"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: checknewlines file.liz...")
		os.Exit(2)
	}
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run checks every file and returns the exit status.
func run(paths []string, out, errOut io.Writer) int {
	status, changes, files := 0, 0, 0
	for _, path := range paths {
		source, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintln(errOut, err)
			status = 2
			continue
		}
		found := lineChanges(string(source))
		for _, change := range found {
			fmt.Fprintf(out, "%s: %s\n", path, change)
		}
		if len(found) > 0 {
			changes += len(found)
			files++
		}
	}

	if changes > 0 {
		fmt.Fprintf(out, "%d lines to change in %d files\n", changes, files)
		if status == 0 {
			status = 1
		}
	}
	return status
}

// lineChanges parses source and returns the lines that LEP 0001 changed.
func lineChanges(source string) []string {
	p := parser.New(lexer.New(source))
	p.ParseProgram()
	return p.LineChanges()
}
