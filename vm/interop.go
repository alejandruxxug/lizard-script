package vm

import (
	"lizardscript/code"
	"lizardscript/evaluator"
	"lizardscript/object"
	"lizardscript/token"
)

// The meeting points with the evaluator. A VM closure is an
// object.Callable, so the evaluator and the builtins call it like their own
// functions; and anything the VM calls that is not one of its closures (a
// builtin, a function of the evaluator) goes through evaluator.ApplyFunction.
// Both count the open calls of both engines for the call depth limit.

// Closure is a function value of the VM: a compiled function, the cells it
// captured and the global environment it was made in. To a program it is a
// FUNCTION like the evaluator's, shown by its source, and only equal to
// itself. The evaluator calls it through object.Callable.
type Closure struct {
	Fn      *code.Function
	Free    []*Cell
	Globals *object.Environment
}

func (cl *Closure) Type() object.ObjectType { return object.FUNCTION_OBJ }

// Inspect shows the source the way object.Function does.
func (cl *Closure) Inspect() string {
	lit := cl.Fn.Literal
	if lit == nil {
		return "program"
	}
	return (&object.Function{Parameters: lit.Parameters, Defaults: lit.Defaults, Body: lit.Body}).Inspect()
}

// ParameterCount is the number of parameters, defaults included.
func (cl *Closure) ParameterCount() int {
	return cl.Fn.NumParams
}

// Call runs the closure with args, for the evaluator or a builtin, with the
// checks of a call written in a program: the step, the arguments and the
// call depth (which counts the calls of both engines).
func (cl *Closure) Call(tok token.Token, args []object.Object) object.Object {
	if err := evaluator.CountStep(tok); err != nil {
		return err
	}
	if err := arityError(tok, cl.Fn, len(args)); err != nil {
		return err
	}
	depth := evaluator.CallDepth()
	if depth >= evaluator.MaxCallDepth {
		return overflow(tok)
	}

	vm := getVM()
	defer putVM(vm)
	vm.depthBase = depth
	vm.start(cl, args)
	return vm.run()
}

// arityError is the error of a call with the wrong number of arguments, the
// same as the evaluator's, or nil.
func arityError(tok token.Token, fn *code.Function, given int) *object.Error {
	if given >= fn.Required && given <= fn.NumParams {
		return nil
	}
	if fn.Required == fn.NumParams {
		return evaluator.NewError(tok, "wrong number of arguments: want %d, got %d", fn.Required, given)
	}
	return evaluator.NewError(tok, "wrong number of arguments: want %d to %d, got %d",
		fn.Required, fn.NumParams, given)
}

func overflow(tok token.Token) *object.Error {
	err := evaluator.NewError(tok, "stack overflow: more than %d nested calls", evaluator.MaxCallDepth)
	err.Uncatchable = true
	return err
}

// callForeign calls anything that is not a closure of the VM (a builtin, a
// function of the evaluator) through the evaluator, which counts the open
// calls of this run too.
func (vm *VM) callForeign(tok int, unit *code.Unit, function object.Object, args []object.Object) object.Object {
	saved := evaluator.CallDepth()
	evaluator.SetCallDepth(vm.depthBase + len(vm.frames))
	var result object.Object
	if p := profiler; p != nil {
		p.enterForeign(function)
		result = evaluator.ApplyFunction(unit.Tokens[tok], function, args)
		p.leave()
	} else {
		result = evaluator.ApplyFunction(unit.Tokens[tok], function, args)
	}
	evaluator.SetCallDepth(saved)
	return result
}
