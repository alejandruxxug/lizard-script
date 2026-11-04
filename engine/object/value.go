package object

import "math"

// Value is a value the way the bytecode VM holds it: on its stack, in its
// variables and cells, and in the constants of compiled code. An integer or
// a float is kept inside the Value, as the bits of the number, so a number
// the VM computes and stores makes no object and no garbage. Anything else
// (a string, an array, a hash, a function, null, true and false) is kept as
// its Object.
//
// A number becomes an *Integer or a *Float ("boxed", with Object) only where
// it leaves the VM: given to a builtin or to the evaluator, stored in an
// array, a hash or a point, or written to an Environment. It comes back
// unboxed with ValueOf, which keeps the box it came in: a number read from
// an array or returned by a builtin goes on to the next builtin with no new
// object. A number is always known by its kind, one byte to compare, never
// by its object.
//
// The zero Value holds nothing (IsEmpty): the VM's "not set yet" for a
// variable, like a nil Object.
type Value struct {
	kind ValueKind
	bits uint64 // the integer, or the float's bits
	obj  Object // the object, or a number's box when it came in one
}

// ValueKind says what a Value holds.
type ValueKind uint8

const (
	KindObject ValueKind = iota // an Object, or nothing
	KindInt                     // an integer, in bits
	KindFloat                   // a float, in bits
)

// IntValue is the Value of an integer.
func IntValue(i int64) Value {
	return Value{kind: KindInt, bits: uint64(i)}
}

// FloatValue is the Value of a float.
func FloatValue(f float64) Value {
	return Value{kind: KindFloat, bits: math.Float64bits(f)}
}

// ValueOfInteger and ValueOfFloat are the Values of an integer and a float
// object, which they keep as the number's box.
func ValueOfInteger(i *Integer) Value {
	return Value{kind: KindInt, bits: uint64(i.Value), obj: i}
}

func ValueOfFloat(f *Float) Value {
	return Value{kind: KindFloat, bits: math.Float64bits(f.Value), obj: f}
}

// ValueOf is the Value of obj, unboxed when it is a number.
func ValueOf(obj Object) Value {
	switch o := obj.(type) {
	case *Integer:
		return ValueOfInteger(o)
	case *Float:
		return ValueOfFloat(o)
	}
	return Value{obj: obj}
}

// ObjectValue is the Value of obj, which the caller knows is not a number:
// a string, a function, a boolean, null.
func ObjectValue(obj Object) Value {
	return Value{obj: obj}
}

// Kind is what v holds.
func (v Value) Kind() ValueKind { return v.kind }

// IsInt reports whether v is an integer.
func (v Value) IsInt() bool { return v.kind == KindInt }

// IsFloat reports whether v is a float.
func (v Value) IsFloat() bool { return v.kind == KindFloat }

// IsNumber reports whether v is an integer or a float.
func (v Value) IsNumber() bool { return v.kind != KindObject }

// IsEmpty reports whether v holds nothing at all.
func (v Value) IsEmpty() bool { return v.kind == KindObject && v.obj == nil }

// Int is the integer of v, which must be one.
func (v Value) Int() int64 { return int64(v.bits) }

// Float is the float of v, which must be one.
func (v Value) Float() float64 { return math.Float64frombits(v.bits) }

// Number is v as a float64: its float, or its integer converted. v must be
// a number.
func (v Value) Number() float64 {
	if v.kind == KindInt {
		return float64(int64(v.bits))
	}
	return math.Float64frombits(v.bits)
}

// Ref is the Object of v without boxing, for the code that tests what kind
// of object v holds: nil for an empty Value and for a number made here, the
// box of a number that came in one. Test for numbers first.
func (v Value) Ref() Object { return v.obj }

// Object is v as an Object: a number's box when it came in one, or a new
// one (shared when it is small, see IntegerOf and FloatOf). An empty Value
// gives nil.
func (v Value) Object() Object {
	if v.obj != nil || v.kind == KindObject {
		return v.obj
	}
	if v.kind == KindInt {
		return IntegerOf(int64(v.bits))
	}
	return FloatOf(math.Float64frombits(v.bits))
}
