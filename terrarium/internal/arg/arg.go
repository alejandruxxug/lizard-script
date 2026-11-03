// Package arg has the helpers every Terrarium module uses: checking the
// arguments of a builtin, making values, and building a module hash.
//
// Errors are plain *object.Error values without a position. The evaluator
// adds the position and the builtin's name, so a message reads
// "[line 1, column 8] runtime error: fs.read: want a STRING path, got INTEGER".
package arg

import (
	"fmt"
	"image/color"
	"lizardscript/evaluator"
	"lizardscript/object"
	"math"
	"sort"
)

// Errorf makes the error of a builtin.
func Errorf(format string, args ...any) *object.Error {
	return &object.Error{Message: fmt.Sprintf(format, args...)}
}

// Count checks that there are from min to max arguments. A max of -1 means
// any number from min up.
func Count(args []object.Object, min, max int) *object.Error {
	if len(args) >= min && (max < 0 || len(args) <= max) {
		return nil
	}
	var want string
	switch {
	case max < 0:
		want = fmt.Sprintf("at least %d", min)
	case min == max:
		want = fmt.Sprintf("%d", min)
	case max == min+1:
		want = fmt.Sprintf("%d or %d", min, max)
	default:
		want = fmt.Sprintf("%d to %d", min, max)
	}
	return Errorf("wrong number of arguments: want %s, got %d", want, len(args))
}

// Name is how a message names a value it didn't expect: its type, or for a
// native its kind, like "a window".
func Name(o object.Object) string {
	if n, ok := o.(*object.Native); ok {
		return article(n.Kind) + " " + n.Kind
	}
	return string(o.Type())
}

// Show is a value the way a message shows what it got: a string in
// quotes, anything else as puts shows it.
func Show(o object.Object) string {
	if s, ok := o.(*object.String); ok {
		return object.Quote(s.Value)
	}
	return o.Inspect()
}

// article is "a" or "an" for word.
func article(word string) string {
	if word != "" && (word[0] == 'a' || word[0] == 'e' || word[0] == 'i' || word[0] == 'o' || word[0] == 'u' ||
		word[0] == 'A' || word[0] == 'E' || word[0] == 'I' || word[0] == 'O' || word[0] == 'U') {
		return "an"
	}
	return "a"
}

// wrong is the error of argument i: want an INTEGER width, got STRING.
func wrong(args []object.Object, i int, kind, what string) *object.Error {
	return Errorf("want %s %s %s, got %s", article(kind), kind, what, Name(args[i]))
}

// String reads argument i as a string. what names it in the message.
func String(args []object.Object, i int, what string) (string, *object.Error) {
	s, ok := args[i].(*object.String)
	if !ok {
		return "", wrong(args, i, "STRING", what)
	}
	return s.Value, nil
}

// Int reads argument i as an integer.
func Int(args []object.Object, i int, what string) (int64, *object.Error) {
	n, ok := args[i].(*object.Integer)
	if !ok {
		return 0, wrong(args, i, "INTEGER", what)
	}
	return n.Value, nil
}

// IntIn reads argument i as an integer from low to high.
func IntIn(args []object.Object, i int, what string, low, high int64) (int, *object.Error) {
	n, err := Int(args, i, what)
	if err != nil {
		return 0, err
	}
	if n < low || n > high {
		return 0, Errorf("the %s must be from %d to %d, got %d", what, low, high, n)
	}
	return int(n), nil
}

// Number reads an integer or a float as a float64.
func Number(o object.Object) (float64, bool) {
	switch v := o.(type) {
	case *object.Integer:
		return float64(v.Value), true
	case *object.Float:
		return v.Value, true
	}
	return 0, false
}

// NumberArg reads argument i as a number.
func NumberArg(args []object.Object, i int, what string) (float64, *object.Error) {
	n, ok := Number(args[i])
	if !ok {
		return 0, Errorf("want a number %s, got %s", what, Name(args[i]))
	}
	return n, nil
}

// XY reads a point's first two coordinates.
func XY(o object.Object) (x, y float64, ok bool) {
	p, isPoint := o.(*object.Point)
	if !isPoint || len(p.Coords) < 2 {
		return 0, 0, false
	}
	x, okX := Number(p.Coords[0])
	y, okY := Number(p.Coords[1])
	return x, y, okX && okY
}

// Point reads argument i as a point (x, y).
func Point(args []object.Object, i int, what string) (x, y float64, err *object.Error) {
	x, y, ok := XY(args[i])
	if !ok {
		return 0, 0, wrong(args, i, "POINT", what)
	}
	return x, y, nil
}

// Color reads argument i as a color.
func Color(args []object.Object, i int, what string) (color.NRGBA, *object.Error) {
	c, ok := args[i].(*object.Color)
	if !ok {
		return color.NRGBA{}, wrong(args, i, "COLOR", what)
	}
	return color.NRGBA{R: c.R, G: c.G, B: c.B, A: c.A}, nil
}

// Hash reads argument i as a hash.
func Hash(args []object.Object, i int, what string) (*object.Hash, *object.Error) {
	h, ok := args[i].(*object.Hash)
	if !ok {
		return nil, wrong(args, i, "HASH", what)
	}
	return h, nil
}

// Array reads argument i as an array.
func Array(args []object.Object, i int, what string) (*object.Array, *object.Error) {
	a, ok := args[i].(*object.Array)
	if !ok {
		return nil, wrong(args, i, "ARRAY", what)
	}
	return a, nil
}

// Native reads argument i as a native of kind, and gives its value.
func Native(args []object.Object, i int, kind string) (any, *object.Error) {
	n, ok := args[i].(*object.Native)
	if !ok || n.Kind != kind {
		return nil, Errorf("want %s %s, got %s", article(kind), kind, Name(args[i]))
	}
	return n.Value, nil
}

// Values every module returns.
var (
	Null  object.Object = evaluator.NULL
	True  object.Object = evaluator.TRUE
	False object.Object = evaluator.FALSE
)

// Bool is TRUE or FALSE.
func Bool(b bool) object.Object {
	if b {
		return True
	}
	return False
}

// Str makes a string.
func Str(s string) object.Object { return &object.String{Value: s} }

// Int64 makes an integer.
func Int64(n int64) object.Object { return &object.Integer{Value: n} }

// Float makes a float. A value that isn't finite becomes 0, since a float
// in lizard-script never is.
func Float(f float64) object.Object {
	if math.IsInf(f, 0) || math.IsNaN(f) {
		f = 0
	}
	return &object.Float{Value: f}
}

// PointOf makes the point (x, y) of two integers.
func PointOf(x, y int) object.Object {
	return &object.Point{Coords: []object.Object{Int64(int64(x)), Int64(int64(y))}}
}

// ColorOf makes a color.
func ColorOf(c color.NRGBA) object.Object {
	return &object.Color{R: c.R, G: c.G, B: c.B, A: c.A}
}

// Set puts value under the string key in h.
func Set(h *object.Hash, key string, value object.Object) {
	h.Set(&object.String{Value: key}, value)
}

// Module makes the hash of a module: every function becomes a builtin
// called name.function, and the keys are in alphabetical order, so keys()
// gives the same list every time.
func Module(name string, functions map[string]object.BuiltinFunction) *object.Hash {
	names := make([]string, 0, len(functions))
	for function := range functions {
		names = append(names, function)
	}
	sort.Strings(names)
	h := object.NewHash()
	for _, function := range names {
		Set(h, function, &object.Builtin{Name: name + "." + function, Fn: functions[function]})
	}
	return h
}
