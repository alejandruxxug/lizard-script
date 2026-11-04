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
//	io     the standard input, output and error streams
//	lizard the language itself: tokens, syntax trees, sandboxed runs, docs
//	window a window to draw in, with its input (package terrarium/window)
//	pen    pen pressure, which comes in the window's events
//
// The standard modules (std.go) are bigger libraries in lizard-script that
// a program loads with use, like the flower graphics family (flower/lily2d).
//
// The Go code here only reaches the system: what to draw, and every pixel
// of it, is decided by the program.
package terrarium

import (
	"github.com/alejandruxxug/lizard-script/engine/object"
	"sync"
)

// Args are the program's own arguments, the ones after its file name. A
// host sets them before Install; os.args() gives them.
var Args []string

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
//	import _ "github.com/alejandruxxug/lizard-script/terrarium/window"
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
	env.Define("io", ioModule())
	env.Define("lizard", lizardReflectModule())
	for _, name := range lizardModules {
		module, err := lizardModule(name)
		if err != nil {
			panic(err)
		}
		env.Define(name, module)
	}
}
