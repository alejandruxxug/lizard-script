// Command lizardscript runs lizard-script programs.
//
//	lizardscript                           starts the REPL
//	lizardscript program.liz [args]        runs a file
//	lizardscript --vm program.liz [args]   runs a file on the bytecode VM
//	lizardscript --profile program.liz     runs it on the VM and prints, after,
//	                                       where the time went (to stderr)
//	lizardscript run [--vm] dir [args]     runs the program of a folder, dir/main.liz
//	lizardscript test [--vm] [--bench] [--update] [dir]
//	                                       runs the tests of a folder (see tools/testrunner)
//	lizardscript build [--vm] [--os OS] [--arch ARCH] [--runtime FILE] dir -o FILE
//	                                       makes one executable of a program folder
//	                                       (see tools/build)
//	lizardscript drive [--pid N] Command   sends a command to a running window
//	                                       that has automation on (try List)
//	lizardscript fmt [-w] [--check] path...
//	                                       lays out the code of files and folders
//	                                       one way (see docs/STYLE.md)
//	lizardscript check [--json] path...    finds mistakes in them without running
//	                                       them (see package check)
//	lizardscript lsp                       the language server, for an editor, on
//	                                       the standard input and output (see package lsp)
//
// A program has the Terrarium, the standard library, installed, and
// os.args() gives its args. Its use statements look for modules next to the
// file that says them, then in the folders of $LIZARD_PATH. The REPL has
// all of the Terrarium but window and pen. A program made with build runs
// its own main.liz with every argument.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"lizardscript/bundle"
	"lizardscript/check"
	"lizardscript/evaluator"
	"lizardscript/format"
	"lizardscript/lsp"
	"lizardscript/repl"
	"lizardscript/terrarium"
	"lizardscript/terrarium/window" // adds window and pen to terrarium.Install
	"lizardscript/tools/build"
	"lizardscript/tools/testrunner"
	"lizardscript/vm"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	evaluator.ReadModule = bundle.ReadFile
	evaluator.ModulePath = filepath.SplitList(os.Getenv("LIZARD_PATH"))
	if program := builtProgram(); program != nil {
		os.Exit(runBuilt(program, os.Args[1:]))
	}

	args := os.Args[1:]
	if len(args) == 0 {
		fmt.Println("Welcome to lizard-script 🦎 The language so easy that even a lizard can understand it.")
		fmt.Println("Type some code, or exit to leave.")
		repl.Start(os.Stdin, os.Stdout)
		return
	}
	switch args[0] {
	case "run":
		os.Exit(runCommand(args[1:]))
	case "test":
		os.Exit(testCommand(args[1:]))
	case "build":
		os.Exit(buildCommand(args[1:]))
	case "drive":
		os.Exit(driveCommand(args[1:]))
	case "fmt":
		os.Exit(fmtCommand(args[1:]))
	case "check":
		os.Exit(checkCommand(args[1:]))
	case "lsp":
		os.Exit(lspCommand())
	}

	onVM, profile := false, false
	for len(args) > 0 && (args[0] == "--vm" || args[0] == "--profile") {
		onVM = true
		profile = profile || args[0] == "--profile"
		args = args[1:]
	}
	if len(args) == 0 || len(args[0]) > 1 && args[0][0] == '-' {
		usage()
	}
	if profile {
		vm.StartProfile()
	}
	ok := runFile(args[0], args[1:], onVM)
	if profile {
		result := vm.StopProfile()
		result.File = args[0]
		result.Write(os.Stderr)
	}
	if !ok {
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `usage: lizardscript [--vm | --profile] [file [args...]]
       lizardscript run [--vm] dir [args...]
       lizardscript test [--vm] [--bench] [--update] [dir]
       lizardscript build [--vm] [--os OS] [--arch ARCH] [--runtime FILE] dir -o FILE
       lizardscript drive [--pid N] Command [args...]
       lizardscript fmt [-w] [--check] path...
       lizardscript check [--json] path...
       lizardscript lsp`)
	os.Exit(2)
}

// runFile runs the program in file with args, and reports whether it ran
// without errors.
func runFile(file string, args []string, onVM bool) bool {
	source, err := bundle.ReadFile(file)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return false
	}
	terrarium.Args = args
	return repl.RunFile(filepath.ToSlash(file), string(source), os.Stdout, os.Stderr, onVM)
}

// runCommand is lizardscript run: the folder's main.liz (or a file).
func runCommand(args []string) int {
	onVM := len(args) > 0 && args[0] == "--vm"
	if onVM {
		args = args[1:]
	}
	if len(args) == 0 {
		usage()
	}
	file := args[0]
	if info, err := os.Stat(file); err == nil && info.IsDir() {
		file = filepath.Join(file, "main.liz")
	}
	if !runFile(file, args[1:], onVM) {
		return 1
	}
	return 0
}

// testCommand is lizardscript test.
func testCommand(args []string) int {
	flags := flag.NewFlagSet("test", flag.ContinueOnError)
	onVM := flags.Bool("vm", false, "run the tests on the bytecode VM")
	bench := flags.Bool("bench", false, "run the *_bench.liz benchmarks too")
	update := flags.Bool("update", false, "rewrite each .expected file with the output")
	if err := flags.Parse(args); err != nil || flags.NArg() > 1 {
		usage()
	}
	dir := "."
	if flags.NArg() == 1 {
		dir = flags.Arg(0)
	}
	failed, err := testrunner.Run(testrunner.Options{Dir: dir, VM: *onVM, Bench: *bench, Update: *update, Out: os.Stdout})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	if failed > 0 {
		return 1
	}
	return 0
}

// buildCommand is lizardscript build. The flags may come before or after
// the folder.
func buildCommand(args []string) int {
	flags := flag.NewFlagSet("build", flag.ContinueOnError)
	onVM := flags.Bool("vm", false, "the program runs on the bytecode VM")
	output := flags.String("o", "", "the executable to make")
	goos := flags.String("os", "", "the system to build for: darwin, windows, linux (default: this one)")
	goarch := flags.String("arch", "", "the processor to build for: amd64, arm64 (default: this one)")
	runtimeFile := flags.String("runtime", "", "a lizardscript built for that system, to use")
	var dirs []string
	for len(args) > 0 {
		if err := flags.Parse(args); err != nil {
			usage()
		}
		args = flags.Args()
		if len(args) > 0 {
			dirs = append(dirs, args[0])
			args = args[1:]
		}
	}
	if len(dirs) != 1 || *output == "" {
		usage()
	}
	files, err := build.Build(build.Options{Dir: dirs[0], Output: *output, GOOS: *goos, GOARCH: *goarch, VM: *onVM, Runtime: *runtimeFile})
	if err != nil {
		fmt.Fprintln(os.Stderr, "lizardscript build:", err)
		return 1
	}
	fmt.Printf("built %s with %s\n", *output, strings.Join(files, ", "))
	return 0
}

// driveCommand is lizardscript drive: one command to a window app that has
// automation on.
func driveCommand(args []string) int {
	if err := window.Drive(args, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

// fmtCommand is lizardscript fmt: it prints the formatted code of each
// file, or with -w writes it back, or with --check lists the files that are
// not formatted and ends with 1 if there is any. A file with syntax errors
// is left alone and ends it with 2.
func fmtCommand(args []string) int {
	flags := flag.NewFlagSet("fmt", flag.ContinueOnError)
	write := flags.Bool("w", false, "write the result back to each file")
	list := flags.Bool("check", false, "list the files that are not formatted")
	if err := flags.Parse(args); err != nil || flags.NArg() == 0 {
		usage()
	}
	files, err := lizFiles(flags.Args())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	status := 0
	for _, file := range files {
		source, err := os.ReadFile(file)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			status = 2
			continue
		}
		formatted, err := format.Source(string(source))
		if err != nil {
			for _, line := range strings.Split(err.Error(), "\n") {
				fmt.Fprintf(os.Stderr, "%s: %s\n", filepath.ToSlash(file), line)
			}
			status = 2
			continue
		}
		changed := formatted != string(source)
		switch {
		case *list:
			if changed {
				fmt.Println(filepath.ToSlash(file))
				status = max(status, 1)
			}
		case *write:
			if changed {
				if err := os.WriteFile(file, []byte(formatted), 0o644); err != nil {
					fmt.Fprintln(os.Stderr, err)
					status = 2
				}
			}
		default:
			fmt.Print(formatted)
		}
	}
	return status
}

// checkCommand is lizardscript check: the findings of each file, one per
// line, or as JSON with --json. It ends with 1 when there is an error (a
// warning is not one), 2 when a file can't be read.
func checkCommand(args []string) int {
	flags := flag.NewFlagSet("check", flag.ContinueOnError)
	asJSON := flags.Bool("json", false, "write the findings as a JSON array")
	if err := flags.Parse(args); err != nil || flags.NArg() == 0 {
		usage()
	}
	files, err := lizFiles(flags.Args())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	check.Globals = terrarium.CheckGlobals()

	type finding struct {
		File string `json:"file"`
		check.Diagnostic
	}
	findings := []finding{}
	status := 0
	for _, file := range files {
		diagnostics, err := check.File(file)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			status = 2
			continue
		}
		for _, d := range diagnostics {
			findings = append(findings, finding{filepath.ToSlash(file), d})
			if d.Severity == check.Error && status == 0 {
				status = 1
			}
		}
	}

	if *asJSON {
		out, _ := json.MarshalIndent(findings, "", "  ")
		fmt.Println(string(out))
		return status
	}
	for _, f := range findings {
		fmt.Printf("%s: [line %d, column %d] %s: %s (%s)\n", f.File, f.Line, f.Column, f.Severity, f.Message, f.Rule)
		if f.Hint != "" {
			fmt.Printf("    hint: %s\n", f.Hint)
		}
	}
	return status
}

// lspCommand is lizardscript lsp: the language server on the standard
// input and output, with the Terrarium's names.
func lspCommand() int {
	server := lsp.New(lsp.Options{Globals: terrarium.CheckGlobals(), Debounce: 150 * time.Millisecond})
	if err := server.Serve(os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "lizardscript lsp:", err)
		return 1
	}
	return 0
}

// lizFiles are the files of paths: each file as it is, and the .liz files
// inside each folder and the folders in it, in order.
func lizFiles(paths []string) ([]string, error) {
	var files []string
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			return nil, err
		}
		if !info.IsDir() {
			files = append(files, path)
			continue
		}
		err = filepath.WalkDir(path, func(file string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() && strings.HasSuffix(file, ".liz") {
				files = append(files, file)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return files, nil
}

// builtProgram is the program built into this executable, or nil for a
// plain lizardscript.
func builtProgram() *bundle.Program {
	self, err := os.Executable()
	if err != nil {
		return nil
	}
	program, err := bundle.Open(self)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	return program
}

// runBuilt runs the program built into this executable with args, and
// returns the exit code.
func runBuilt(program *bundle.Program, args []string) int {
	bundle.Current = program
	main := bundle.Path(program.Main)
	if !runFile(main, args, program.VM) {
		return 1
	}
	return 0
}
