// Command lizardscript runs lizard-script programs.
//
//	lizardscript              starts the REPL
//	lizardscript program.liz  runs a file
//	lizardscript --vm program.liz  runs a file on the bytecode VM
package main

import (
	"fmt"
	"lizardscript/repl"
	"os"
)

func main() {
	switch len(os.Args) {
	case 1:
		fmt.Println("Welcome to lizard-script 🦎 The language so easy that even a lizard can understand it.")
		fmt.Println("Type some code, or exit to leave.")
		repl.Start(os.Stdin, os.Stdout)

	case 2:
		source, err := os.ReadFile(os.Args[1])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if !repl.Run(string(source), os.Stdout, os.Stderr) {
			os.Exit(1)
		}

	case 3:
		if os.Args[1] != "--vm" {
			usage()
		}
		source, err := os.ReadFile(os.Args[2])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if !repl.RunVM(string(source), os.Stdout, os.Stderr) {
			os.Exit(1)
		}

	default:
		usage()
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: lizardscript [--vm] [file]")
	os.Exit(2)
}
