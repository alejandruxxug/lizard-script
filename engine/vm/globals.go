package vm

import (
	"github.com/alejandruxxug/lizard-script/engine/code"
	"github.com/alejandruxxug/lizard-script/engine/evaluator"
	"github.com/alejandruxxug/lizard-script/engine/object"
)

// A read of a global or a builtin looks the name up in the environment's
// map, and for a builtin also misses there first. Each read in the code
// remembers what it found (code.Global), and finds it again for free while
// nothing could have changed it: the environment has the same version (no
// let or = on any of its names since) and the libraries were not reset.
//
// The version is one for the whole environment, so in a loop at the top of
// a program that assigns a global each round, every read misses. Writing
// the memory each time would cost more than it saves, so after a few such
// misses in a row a read only writes it once in a while.

const (
	missesBeforeBackoff = 4
	backoff             = 32 // writes the memory one miss in this many
)

// global reads a name in the global environment, then the builtins.
func global(n *code.Name, globals *object.Environment) (object.Value, *object.Error) {
	memo := &n.Global
	version, resets := globals.Version(), evaluator.LibraryResets()
	if memo.Env == globals && memo.Resets == resets {
		if memo.Version == version {
			memo.Misses = 0
			return memo.Value, nil
		}
		memo.Misses++
	}
	found, ok := globals.Get(n.Name)
	if !ok {
		found, ok = evaluator.Builtin(n.Name)
	}
	if !ok {
		return object.Value{}, evaluator.NewError(n.Token, "identifier not found: %s", n.Name)
	}
	value := object.ValueOf(found)
	if memo.Misses < missesBeforeBackoff || memo.Misses%backoff == 0 {
		memo.Env, memo.Version, memo.Resets, memo.Value = globals, version, resets, value
	}
	return value, nil
}
