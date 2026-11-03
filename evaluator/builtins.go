package evaluator

import (
	"fmt"
	"io"
	"lizardscript/object"
	"math"
	"os"
	"strconv"
	"strings"
)

// Output is where puts writes. Tests and the REPL can change it.
var Output io.Writer = os.Stdout

// maxRangeLength limits the arrays made by range.
const maxRangeLength = 1 << 24

// builtins are the functions that exist without being declared. A variable
// with the same name hides them.
var builtins = map[string]*object.Builtin{}

func init() {
	for name, fn := range map[string]object.BuiltinFunction{
		"len":    builtinLen,
		"puts":   builtinPuts,
		"push":   builtinPush,
		"pop":    builtinPop,
		"first":  builtinFirst,
		"last":   builtinLast,
		"rest":   builtinRest,
		"keys":   builtinKeys,
		"values": builtinValues,
		"delete": builtinDelete,
		"type":   builtinType,
		"str":    builtinStr,
		"error":  builtinError,
		"int":    builtinInt,
		"float":  builtinFloat,
		"range":  builtinRange,
		"rgb":    builtinRGB,
		"rgba":   builtinRGBA,
		"hsl":    builtinHSL,
		"point":  builtinPoint,
		"same":   builtinSame,
	} {
		builtins[name] = &object.Builtin{Name: name, Fn: fn}
	}
}

// argError is an error of a builtin. applyFunction adds the position and the
// name of the builtin to the message.
func argError(format string, args ...any) *object.Error {
	return &object.Error{Message: fmt.Sprintf(format, args...)}
}

func wrongCount(want string, got int) *object.Error {
	return argError("wrong number of arguments: want %s, got %d", want, got)
}

// len(x) is the number of characters of a string, or of elements of an
// array or hash.
func builtinLen(args ...object.Object) object.Object {
	if len(args) != 1 {
		return wrongCount("1", len(args))
	}
	switch arg := args[0].(type) {
	case *object.String:
		return &object.Integer{Value: int64(arg.Len())}
	case *object.Array:
		return &object.Integer{Value: int64(len(arg.Elements))}
	case *object.Hash:
		return &object.Integer{Value: int64(len(arg.Pairs))}
	default:
		return argError("argument not supported, got %s", arg.Type())
	}
}

// puts(a, b, ...) writes its arguments separated by spaces, then a newline.
func builtinPuts(args ...object.Object) object.Object {
	parts := make([]string, len(args))
	for i, arg := range args {
		parts[i] = arg.Inspect()
	}
	fmt.Fprintln(Output, strings.Join(parts, " "))
	return NULL
}

// arrayArgument checks that a builtin got exactly count arguments and that
// the first one is an array.
func arrayArgument(args []object.Object, count int) (*object.Array, *object.Error) {
	if len(args) != count {
		return nil, wrongCount(strconv.Itoa(count), len(args))
	}
	array, ok := args[0].(*object.Array)
	if !ok {
		return nil, argError("argument must be ARRAY, got %s", args[0].Type())
	}
	return array, nil
}

// push(list, x) adds x at the end of list and returns list.
func builtinPush(args ...object.Object) object.Object {
	array, err := arrayArgument(args, 2)
	if err != nil {
		return err
	}
	array.Elements = append(array.Elements, args[1])
	return array
}

// pop(list) removes the last element of list and returns it.
func builtinPop(args ...object.Object) object.Object {
	array, err := arrayArgument(args, 1)
	if err != nil {
		return err
	}
	if len(array.Elements) == 0 {
		return argError("the array is empty")
	}
	last := array.Elements[len(array.Elements)-1]
	array.Elements = array.Elements[:len(array.Elements)-1]
	return last
}

// first(list) is the first element, or null if list is empty.
func builtinFirst(args ...object.Object) object.Object {
	array, err := arrayArgument(args, 1)
	if err != nil {
		return err
	}
	if len(array.Elements) == 0 {
		return NULL
	}
	return array.Elements[0]
}

// last(list) is the last element, or null if list is empty.
func builtinLast(args ...object.Object) object.Object {
	array, err := arrayArgument(args, 1)
	if err != nil {
		return err
	}
	if len(array.Elements) == 0 {
		return NULL
	}
	return array.Elements[len(array.Elements)-1]
}

// rest(list) is a new array with every element but the first, or null if
// list is empty.
func builtinRest(args ...object.Object) object.Object {
	array, err := arrayArgument(args, 1)
	if err != nil {
		return err
	}
	if len(array.Elements) == 0 {
		return NULL
	}
	elements := make([]object.Object, len(array.Elements)-1)
	copy(elements, array.Elements[1:])
	return &object.Array{Elements: elements}
}

// hashArgument checks that a builtin got exactly count arguments and that
// the first one is a hash.
func hashArgument(args []object.Object, count int) (*object.Hash, *object.Error) {
	if len(args) != count {
		return nil, wrongCount(strconv.Itoa(count), len(args))
	}
	hash, ok := args[0].(*object.Hash)
	if !ok {
		return nil, argError("argument must be HASH, got %s", args[0].Type())
	}
	return hash, nil
}

// keys(hash) is an array with the keys, in the order they were added.
func builtinKeys(args ...object.Object) object.Object {
	hash, err := hashArgument(args, 1)
	if err != nil {
		return err
	}
	keys := []object.Object{}
	hash.Each(func(pair object.HashPair) { keys = append(keys, pair.Key) })
	return &object.Array{Elements: keys}
}

// values(hash) is an array with the values, in the order of keys(hash).
func builtinValues(args ...object.Object) object.Object {
	hash, err := hashArgument(args, 1)
	if err != nil {
		return err
	}
	values := []object.Object{}
	hash.Each(func(pair object.HashPair) { values = append(values, pair.Value) })
	return &object.Array{Elements: values}
}

// delete(hash, key) removes key from hash, and returns whether it was there.
func builtinDelete(args ...object.Object) object.Object {
	hash, err := hashArgument(args, 2)
	if err != nil {
		return err
	}
	key, ok := args[1].(object.Hashable)
	if !ok {
		return argError("unusable as hash key: %s", args[1].Type())
	}
	return nativeBoolToBooleanObject(hash.Delete(key))
}

// same(a, b) reports whether a and b are the same value: the same array,
// hash, function or native, not two with equal contents. Numbers, strings,
// booleans, colors, points and null have no identity of their own, so for
// them it is ===.
func builtinSame(args ...object.Object) object.Object {
	if len(args) != 2 {
		return wrongCount("2", len(args))
	}
	a, b := args[0], args[1]
	switch a.(type) {
	case *object.Array, *object.Hash, *object.Function, *object.Builtin, *object.Native, object.Callable:
		return nativeBoolToBooleanObject(a == b)
	}
	return nativeBoolToBooleanObject(strictEquals(a, b))
}

// type(x) is the name of the type of x, like "INTEGER".
func builtinType(args ...object.Object) object.Object {
	if len(args) != 1 {
		return wrongCount("1", len(args))
	}
	return &object.String{Value: string(args[0].Type())}
}

// str(x) turns x into a string, the way puts would show it.
func builtinStr(args ...object.Object) object.Object {
	if len(args) != 1 {
		return wrongCount("1", len(args))
	}
	return &object.String{Value: args[0].Inspect()}
}

// error(message) stops the program with a runtime error. The message is a
// string; any other value is turned into one the way str does it. The
// position is added where the call is made. error(message, options) takes
// a hash of options: {"at": "caller"} reports it where the program called
// into this file (see caller.go).
func builtinError(args ...object.Object) object.Object {
	if len(args) == 2 {
		if options, ok := args[1].(*object.Hash); ok {
			atCaller, bad := errorOptions(options)
			if bad != nil {
				return bad
			}
			err := argError("%s", args[0].Inspect())
			err.AtCaller = atCaller
			return err
		}
	}
	if len(args) != 1 {
		// A misuse of error itself is reported like any other builtin.
		return argError("error: wrong number of arguments: want 1, got %d", len(args))
	}
	return argError("%s", args[0].Inspect())
}

// int(x) turns a string with a whole number, a boolean or a float into an
// integer. A float loses its decimals, rounding towards zero.
func builtinInt(args ...object.Object) object.Object {
	if len(args) != 1 {
		return wrongCount("1", len(args))
	}
	switch arg := args[0].(type) {
	case *object.Integer:
		return arg
	case *object.Float:
		integer, ok := floatToInteger(math.Trunc(arg.Value))
		if !ok {
			return argError("the float is too big for an integer: %s", arg.Inspect())
		}
		return integer
	case *object.Boolean:
		if arg.Value {
			return &object.Integer{Value: 1}
		}
		return &object.Integer{Value: 0}
	case *object.String:
		number, err := strconv.ParseInt(arg.Value, 10, 64)
		if err != nil {
			return argError("not a whole number: %q", arg.Value)
		}
		return &object.Integer{Value: number}
	default:
		return argError("argument not supported, got %s", arg.Type())
	}
}

// range(end) is [0, 1, ..., end - 1], and range(start, end) is
// [start, ..., end - 1]. It is empty when end is not after start.
func builtinRange(args ...object.Object) object.Object {
	if len(args) != 1 && len(args) != 2 {
		return wrongCount("1 or 2", len(args))
	}
	bounds := make([]int64, len(args))
	for i, arg := range args {
		integer, ok := arg.(*object.Integer)
		if !ok {
			return argError("arguments must be INTEGER, got %s", arg.Type())
		}
		bounds[i] = integer.Value
	}

	start, end := int64(0), bounds[0]
	if len(bounds) == 2 {
		start, end = bounds[0], bounds[1]
	}
	if end <= start {
		return &object.Array{Elements: []object.Object{}}
	}
	if uint64(end-start) > maxRangeLength {
		return argError("range too long: more than %d elements", maxRangeLength)
	}

	elements := make([]object.Object, 0, end-start)
	for i := start; i < end; i++ {
		elements = append(elements, &object.Integer{Value: i})
	}
	return &object.Array{Elements: elements}
}
