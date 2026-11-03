// Package terrarium is lizard-script's standard library, the Terrarium:
// where lizards live. Its modules are hashes of builtins, used with a dot
// like math is:
//
//	fs     files and folders
//	os     the program's arguments, environment and system
//	time   clocks and waiting
//	json   reading and writing JSON (written in lizard-script)
//	img    images: load, save and change pixels
//	clip   the system clipboard
//	window a window to draw in, with its input (package terrarium/window)
//	pen    pen pressure, which comes in the window's events
//
// The Go code here only reaches the system: what to draw, and every pixel
// of it, is decided by the program.
package terrarium

import (
	"embed"
	"fmt"
	"lizardscript/evaluator"
	"lizardscript/lexer"
	"lizardscript/object"
	"lizardscript/parser"
	"strings"
	"sync"
)

// Args are the program's own arguments, the ones after its file name. A
// host sets them before Install; os.args() gives them.
var Args []string

// sources are the modules written in lizard-script, run into the
// environment by InstallCore.
//
//go:embed lib/*.liz
var sources embed.FS

// lizardModules are the files of sources, in the order they run.
var lizardModules = []string{"json.liz"}

// extras are the modules that live in their own packages, like window,
// added by Register.
var (
	extrasMu sync.Mutex
	extras   []func(env *object.Environment)
)

// Register adds a module that lives in its own package to Install. The
// package calls it from its init, so a host gets that module by importing
// the package, and a host that doesn't import it doesn't link it:
//
//	import _ "lizardscript/terrarium/window"
func Register(install func(env *object.Environment)) {
	extrasMu.Lock()
	defer extrasMu.Unlock()
	extras = append(extras, install)
}

// Install defines every module of the Terrarium in env: the core ones and
// the ones registered by the packages the host imports (window and pen).
// A program's own let with the same name hides a module.
func Install(env *object.Environment) {
	InstallCore(env)
	extrasMu.Lock()
	defer extrasMu.Unlock()
	for _, install := range extras {
		install(env)
	}
}

// InstallCore defines every module but window and pen in env: the ones a
// program can use with no screen, like a REPL or a server.
//
// The modules written in lizard-script are part of the binary and tested
// with it, so one that fails to run is a bug in the Terrarium: it panics.
func InstallCore(env *object.Environment) {
	env.Define("fs", fsModule())
	env.Define("os", osModule())
	env.Define("time", timeModule())
	env.Define("img", imgModule())
	env.Define("clip", clipModule())
	for _, name := range lizardModules {
		if err := runModule(name, env); err != nil {
			panic(err)
		}
	}
}

// runModule runs one module written in lizard-script into env.
func runModule(name string, env *object.Environment) error {
	code, err := sources.ReadFile("lib/" + name)
	if err != nil {
		return err
	}
	p := parser.New(lexer.NewNamed(string(code), "terrarium/"+name))
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		return fmt.Errorf("terrarium/%s: %s", name, strings.Join(p.Errors(), "; "))
	}
	if result, ok := evaluator.Eval(program, env).(*object.Error); ok {
		return fmt.Errorf("terrarium/%s: %s", name, result.Message)
	}
	return nil
}
