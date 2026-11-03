package evaluator

import (
	"lizardscript/object"
	"sort"
)

// setLibrary adds the functions of a library (math, text, list, hash) to
// its hash in the order of their names. A Go map has no order, so filling
// the hash straight from one would print the library differently on every
// run.
func setLibrary(hash *object.Hash, prefix string, functions map[string]object.BuiltinFunction) {
	names := make([]string, 0, len(functions))
	for name := range functions {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		hash.Set(&object.String{Value: name}, &object.Builtin{Name: prefix + name, Fn: functions[name]})
	}
}
