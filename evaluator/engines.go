package evaluator

import (
	"lizardscript/ast"
	"lizardscript/object"
	"lizardscript/token"
	"sort"
)

// The functions in this file let another engine (the bytecode VM in package
// vm) run programs with exactly the rules of this one: it calls these for
// everything but its fast paths, so an operator, an index or a call gives the
// same value and the same error with both. They only wrap what Eval uses.

// MaxCallDepth is how many function calls can be open at once, counting the
// calls of every engine.
const MaxCallDepth = maxCallDepth

// ApplyFunction calls function with args, as a call at tok: a Function, a
// Builtin or a Callable of another engine. Anything else is the error
// "not a function". It counts the step and checks the arguments and the call
// depth, like a call written in a program.
func ApplyFunction(tok token.Token, function object.Object, args []object.Object) object.Object {
	return applyFunction(tok, function, args)
}

// CountStep takes one step at tok: it returns the step-limit error once the
// program has gone over MaxSteps, and nil otherwise.
func CountStep(tok token.Token) *object.Error {
	return countStep(tok)
}

// CallDepth is the number of function calls open right now. An engine that
// runs calls of its own adds them before it calls back into this one, with
// SetCallDepth, so the limit counts all of them.
func CallDepth() int {
	return callDepth
}

// SetCallDepth sets the number of open calls; see CallDepth.
func SetCallDepth(n int) {
	callDepth = n
}

// Infix applies a binary operator other than && and || to two values, with
// errors at tok.
func Infix(tok token.Token, operator string, left, right object.Object) object.Object {
	return evalInfixValues(tok, operator, left, right)
}

// Prefix applies the operator of node to right, the value of its operand.
func Prefix(node *ast.PrefixExpression, right object.Object) object.Object {
	return evalPrefixExpression(node, right)
}

// Index reads left[index], with errors at tok.
func Index(tok token.Token, left, index object.Object) object.Object {
	return evalIndexExpression(tok, left, index)
}

// SetIndex does container[index] = value, with errors at tok.
func SetIndex(tok token.Token, container, index, value object.Object) object.Object {
	return assignIndex(tok, container, index, value)
}

// IsTruthy reports whether a value counts as true in a condition.
func IsTruthy(obj object.Object) bool {
	return isTruthy(obj)
}

// NativeBool returns the TRUE or FALSE object.
func NativeBool(value bool) *object.Boolean {
	return nativeBoolToBooleanObject(value)
}

// Builtin finds a name that no variable has: a builtin, null, or the math,
// text, list and hash libraries, in that order, like a name in a program.
func Builtin(name string) (object.Object, bool) {
	if builtin, ok := builtins[name]; ok {
		return builtin, true
	}
	switch name {
	case "null":
		return NULL, true
	case "math":
		return mathLibrary(), true
	case "text":
		return textLibrary(), true
	case "list":
		return listLibrary(), true
	case "hash":
		return hashLibrary(), true
	}
	return nil, false
}

// BuiltinNames are the names Builtin finds, sorted: the builtins, null and
// the libraries. A tool that checks programs (package check) knows them
// from here.
func BuiltinNames() []string {
	names := []string{"null", "math", "text", "list", "hash"}
	for name := range builtins {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// LibraryResets changes each time ResetMath drops the libraries, the only
// time what Builtin finds for a name can change.
func LibraryResets() uint64 {
	return libraryResets
}

// NewError builds a runtime error at the position of tok.
func NewError(tok token.Token, format string, args ...any) *object.Error {
	return newError(tok, format, args...)
}

// ErrorHash is the value a catch block gets for err.
func ErrorHash(err *object.Error) *object.Hash {
	return errorToHash(err)
}

// IsNumber reports whether obj is an integer or a float.
func IsNumber(obj object.Object) bool {
	return isNumber(obj)
}
