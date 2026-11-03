package vm

import (
	"lizardscript/code"
	"lizardscript/evaluator"
	"lizardscript/object"
)

// Names that are not plain locals: captured variables that may not be set
// yet, and globals. The compiler decided the order to try them in (see
// compiler/scopes.go); this follows it at run time.

// lookup reads a name through n: the captured variables that might be set,
// then the one that surely is, or the globals and the builtins (see
// globals.go).
func lookup(n *code.Name, free []*Cell, globals *object.Environment) (object.Object, *object.Error) {
	for _, i := range n.Checked {
		if value := free[i].Value; value != nil {
			return value, nil
		}
	}
	if n.Final >= 0 {
		return free[n.Final].Value, nil
	}
	return global(n, globals)
}

// assign does name = value through n, like lookup finds it. Builtins can't
// be assigned.
func assign(n *code.Name, value object.Object, free []*Cell, globals *object.Environment) *object.Error {
	for _, i := range n.Checked {
		if free[i].Value != nil {
			free[i].Value = value
			return nil
		}
	}
	if n.Final >= 0 {
		free[n.Final].Value = value
		return nil
	}
	if !globals.Assign(n.Name, value) {
		return evaluator.NewError(n.Token, "can't assign to %s: it was never declared, use let %s = ...",
			n.Name, n.Name)
	}
	return nil
}
