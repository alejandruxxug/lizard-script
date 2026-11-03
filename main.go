// Command lizardscript runs lizard-script programs.
//
//	lizardscript                           starts the REPL
//	lizardscript program.liz [args]        runs a file
//	lizardscript --vm program.liz [args]   runs a file on the bytecode VM
//	lizardscript --profile program.liz     runs it on the VM and prints, after,
//	                                       where the time went (to stderr)
//
// A program has the Terrarium, the standard library, installed, and
// os.args() gives its args. The REPL has all of it but window and pen.
package main

import (
	"fmt"
	"lizardscript/repl"
	"lizardscript/terrarium"
	_ "lizardscript/terrarium/window" // adds window and pen to terrarium.Install
	"lizardscript/vm"
	"os"
)

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		fmt.Println("Welcome to lizard-script 🦎 The language so easy that even a lizard can understand it.")
		fmt.Println("Type some code, or exit to leave.")
		repl.Start(os.Stdin, os.Stdout)
		return
	}

	run, profile := repl.Run, false
	for len(args) > 0 && (args[0] == "--vm" || args[0] == "--profile") {
		run = repl.RunVM
		profile = profile || args[0] == "--profile"
		args = args[1:]
	}
	if len(args) == 0 || len(args[0]) > 1 && args[0][0] == '-' {
		usage()
	}
	terrarium.Args = args[1:]
	source, err := os.ReadFile(args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if profile {
		vm.StartProfile()
	}
	ok := run(string(source), os.Stdout, os.Stderr)
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
	fmt.Fprintln(os.Stderr, "usage: lizardscript [--vm | --profile] [file [args...]]")
	os.Exit(2)
}
