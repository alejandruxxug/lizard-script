package evaluator

import (
	"lizardscript/object"
	"math"
	"math/rand"
	"strconv"
)

// defaultSeed is the seed of math.random at the start of every run, so the
// same program always makes the same picture.
const defaultSeed = 1

// random is where math.random gets its numbers from.
var random = rand.New(rand.NewSource(defaultSeed))

// mathHash is the value of the name math in the current run. It is made the
// first time a program uses it, and not shared between runs, so a program
// that changes it (math.pi = 3) can't affect the next one.
var mathHash *object.Hash

// ResetMath starts a new run for the math library: math.random goes back to
// its default seed, and the math value is made again. A host that runs
// programs one after another calls it before each run, next to ResetSteps.
func ResetMath() {
	random = rand.New(rand.NewSource(defaultSeed))
	mathHash = nil
}

// mathLibrary returns the value of the name math, a hash of builtins and
// constants. A variable called math hides it, like any builtin.
func mathLibrary() *object.Hash {
	if mathHash != nil {
		return mathHash
	}
	hash := object.NewHash()
	for name, fn := range map[string]object.BuiltinFunction{
		"sin":    floatFunction(math.Sin),
		"cos":    floatFunction(math.Cos),
		"tan":    floatFunction(math.Tan),
		"sqrt":   mathSqrt,
		"atan2":  mathAtan2,
		"abs":    mathAbs,
		"floor":  roundingFunction(math.Floor),
		"ceil":   roundingFunction(math.Ceil),
		"round":  roundingFunction(math.Round),
		"min":    mathExtreme(func(a, b float64) bool { return b < a }),
		"max":    mathExtreme(func(a, b float64) bool { return b > a }),
		"random": mathRandom,
		"seed":   mathSeed,
		"dist":   mathDist,
		"angle":  mathAngle,
	} {
		hash.Set(&object.String{Value: name}, &object.Builtin{Name: "math." + name, Fn: fn})
	}
	hash.Set(&object.String{Value: "pi"}, &object.Float{Value: math.Pi})
	hash.Set(&object.String{Value: "tau"}, &object.Float{Value: 2 * math.Pi})
	mathHash = hash
	return hash
}

// numberArguments checks that a builtin got exactly count arguments and that
// all of them are numbers, and returns them as floats.
func numberArguments(args []object.Object, count int) ([]float64, *object.Error) {
	if len(args) != count {
		return nil, wrongCount(strconv.Itoa(count), len(args))
	}
	numbers := make([]float64, count)
	for i, arg := range args {
		if !isNumber(arg) {
			return nil, argError("arguments must be INTEGER or FLOAT, got %s", arg.Type())
		}
		numbers[i] = toFloat(arg)
	}
	return numbers, nil
}

// finite wraps result as a float, or fails when it is infinite or not a
// number.
func finite(result float64) object.Object {
	if math.IsInf(result, 0) || math.IsNaN(result) {
		return argError(notANumber)
	}
	return &object.Float{Value: result}
}

// floatFunction makes a builtin of one number that gives a float, like sin.
func floatFunction(f func(float64) float64) object.BuiltinFunction {
	return func(args ...object.Object) object.Object {
		numbers, err := numberArguments(args, 1)
		if err != nil {
			return err
		}
		return finite(f(numbers[0]))
	}
}

// math.sqrt(x) is the square root of x. A negative x is an error.
func mathSqrt(args ...object.Object) object.Object {
	numbers, err := numberArguments(args, 1)
	if err != nil {
		return err
	}
	if numbers[0] < 0 {
		return argError("can't take the square root of a negative number: %s",
			args[0].Inspect())
	}
	return finite(math.Sqrt(numbers[0]))
}

// math.atan2(y, x) is the angle of the point (x, y), in radians.
func mathAtan2(args ...object.Object) object.Object {
	numbers, err := numberArguments(args, 2)
	if err != nil {
		return err
	}
	return finite(math.Atan2(numbers[0], numbers[1]))
}

// math.abs(x) is x without its sign, an integer for an integer.
func mathAbs(args ...object.Object) object.Object {
	numbers, err := numberArguments(args, 1)
	if err != nil {
		return err
	}
	if integer, ok := args[0].(*object.Integer); ok {
		if integer.Value < 0 {
			return &object.Integer{Value: -integer.Value}
		}
		return integer
	}
	return &object.Float{Value: math.Abs(numbers[0])}
}

// roundingFunction makes floor, ceil or round: a builtin of one number that
// gives an integer.
func roundingFunction(round func(float64) float64) object.BuiltinFunction {
	return func(args ...object.Object) object.Object {
		numbers, err := numberArguments(args, 1)
		if err != nil {
			return err
		}
		if integer, ok := args[0].(*object.Integer); ok {
			return integer
		}
		integer, ok := floatToInteger(round(numbers[0]))
		if !ok {
			return argError("the result is too big for an integer: %s", args[0].Inspect())
		}
		return integer
	}
}

// mathExtreme makes min or max: better says whether the second number beats
// the first. It gives back the winning argument itself, so the minimum of
// two integers is an integer, and the first one wins a tie.
func mathExtreme(better func(first, second float64) bool) object.BuiltinFunction {
	return func(args ...object.Object) object.Object {
		if len(args) < 2 {
			return wrongCount("at least 2", len(args))
		}
		numbers, err := numberArguments(args, len(args))
		if err != nil {
			return err
		}
		winner := 0
		for i := 1; i < len(numbers); i++ {
			if better(numbers[winner], numbers[i]) {
				winner = i
			}
		}
		return args[winner]
	}
}

// math.random() is a float from 0 up to but not including 1, and
// math.random(a, b) is an integer from a to b, both included.
func mathRandom(args ...object.Object) object.Object {
	switch len(args) {
	case 0:
		return &object.Float{Value: random.Float64()}
	case 2:
		low, lowOk := args[0].(*object.Integer)
		high, highOk := args[1].(*object.Integer)
		if !lowOk || !highOk {
			return argError("arguments must be INTEGER, got %s and %s", args[0].Type(), args[1].Type())
		}
		if high.Value < low.Value {
			return argError("the range is empty: %d is less than %d", high.Value, low.Value)
		}
		// The size of the range as an unsigned number, so a range as wide as
		// the integers themselves doesn't overflow. It is 0 for all of them.
		size := uint64(high.Value) - uint64(low.Value) + 1
		offset := random.Uint64()
		if size != 0 {
			offset %= size
		}
		return &object.Integer{Value: int64(uint64(low.Value) + offset)}
	default:
		return wrongCount("0 or 2", len(args))
	}
}

// math.seed(n) restarts math.random from the seed n, so the same seed always
// gives the same numbers.
func mathSeed(args ...object.Object) object.Object {
	if len(args) != 1 {
		return wrongCount("1", len(args))
	}
	seed, ok := args[0].(*object.Integer)
	if !ok {
		return argError("argument must be INTEGER, got %s", args[0].Type())
	}
	random = rand.New(rand.NewSource(seed.Value))
	return NULL
}
