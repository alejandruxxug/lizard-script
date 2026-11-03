package evaluator

import (
	"fmt"
	"lizardscript/object"
	"lizardscript/token"
	"strings"
)

// Errors at the caller. error(message, {"at": "caller"}) is for a library's
// checks of its arguments: the mistake is in the program that called the
// library, so the error is reported at that program's call, not at the
// line of the library that found it. The error keeps its own position
// until it leaves a call made from another file than the one that raised
// it, the call into that file, and takes that call's position and source
// then. The standard modules count as one file for this, since a standard
// library is often several files that call each other: an error one of
// them raises moves to the call from the program. A try inside the file
// that raised it sees it where it was raised.

// AtCaller moves err to call, the call err is leaving, when err asks to be
// reported at the caller and call was made from another file: err takes
// the call's position and source, and is an ordinary error from then on.
// It reports whether err moved.
func AtCaller(err *object.Error, call token.Token) bool {
	if !err.AtCaller || call.Line == 0 || sameLibrary(call.Source, err.Source) {
		return false
	}
	prefix := fmt.Sprintf("[line %d, column %d] runtime error: ", err.Line, err.Column)
	moved := newError(call, "%s", strings.TrimPrefix(err.Message, prefix))
	err.Message, err.Line, err.Column, err.Source = moved.Message, moved.Line, moved.Column, moved.Source
	err.AtCaller = false
	return true
}

// errorOptions reads the options of error(message, options): "at" is
// "caller" or "here" (the default). It gives whether the error is at the
// caller, or the error of a wrong option.
func errorOptions(options *object.Hash) (atCaller bool, bad *object.Error) {
	options.Each(func(pair object.HashPair) {
		if bad != nil {
			return
		}
		key, ok := pair.Key.(*object.String)
		if !ok || key.Value != "at" {
			bad = argError("error: unknown option %s, the only option is at", shown(pair.Key))
			return
		}
		at, ok := pair.Value.(*object.String)
		if !ok || (at.Value != "caller" && at.Value != "here") {
			bad = argError(`error: the option at must be "caller" or "here", got %s`, shown(pair.Value))
			return
		}
		atCaller = at.Value == "caller"
	})
	return atCaller, bad
}

// shown is a value the way a message shows it: a string in quotes.
func shown(o object.Object) string {
	if s, ok := o.(*object.String); ok {
		return object.Quote(s.Value)
	}
	return o.Inspect()
}

// sameLibrary reports whether the sources a and b are one file for
// AtCaller: the same file, or both standard modules.
func sameLibrary(a, b string) bool {
	return a == b || (IsStandardModule(a) && IsStandardModule(b))
}
