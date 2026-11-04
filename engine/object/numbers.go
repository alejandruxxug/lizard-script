package object

import "math"

// Integers and floats can't change, so the small whole numbers that
// programs make all the time (coordinates, counters, rounded values) are
// made once and shared. Sharing them is invisible to a program: === and
// == compare numbers by value. Code that makes many numbers, like the
// bytecode VM's arithmetic, uses these instead of allocating each one.

// The range of the shared numbers, from SmallMin up to but not including
// SmallMax: pixel coordinates of a large canvas fit.
const (
	SmallMin = -1024
	SmallMax = 8192
)

var (
	smallIntegers = make([]Integer, SmallMax-SmallMin)
	smallFloats   = make([]Float, SmallMax-SmallMin)
)

func init() {
	for i := range smallIntegers {
		smallIntegers[i].Value = int64(i + SmallMin)
		smallFloats[i].Value = float64(i + SmallMin)
	}
}

// IntegerOf is the integer value, shared when it is small.
func IntegerOf(value int64) *Integer {
	if value >= SmallMin && value < SmallMax {
		return &smallIntegers[value-SmallMin]
	}
	return &Integer{Value: value}
}

// FloatOf is the float value, shared when it is a small whole number (but
// not -0.0, which prints differently from 0.0).
func FloatOf(value float64) *Float {
	if value >= SmallMin && value < SmallMax && value == math.Trunc(value) && !(value == 0 && math.Signbit(value)) {
		return &smallFloats[int(value)-SmallMin]
	}
	return &Float{Value: value}
}
