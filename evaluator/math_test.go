package evaluator

import (
	"strings"
	"testing"
)

func TestMathFunctions(t *testing.T) {
	ResetMath()
	checkAll(t, []expectation{
		{"math.sin(0)", "0.0"},
		{"math.cos(0)", "1.0"},
		{"math.tan(0)", "0.0"},
		{"math.sin(90deg)", "1.0"},
		{"math.cos(180deg)", "-1.0"},
		{"math.atan2(1, 1) == 45deg", "true"},
		{"math.atan2(0, -1) == math.pi", "true"},
		{"math.atan2(0.0, 1)", "0.0"},
		{"math.sqrt(16)", "4.0"},
		{"math.sqrt(2.25)", "1.5"},
		{"math.sqrt(0)", "0.0"},
		{"math.abs(-3)", "3"},
		{"math.abs(3)", "3"},
		{"math.abs(-1.5)", "1.5"},
		{"math.abs(0)", "0"},
		{"type(math.abs(-3))", "\"INTEGER\""},
		{"type(math.abs(-3.0))", "\"FLOAT\""},
		{"math.floor(1.9)", "1"},
		{"math.floor(-1.1)", "-2"},
		{"math.floor(2)", "2"},
		{"math.ceil(1.1)", "2"},
		{"math.ceil(-1.9)", "-1"},
		{"math.ceil(2.0)", "2"},
		{"math.round(1.5)", "2"},
		{"math.round(2.5)", "3"},
		{"math.round(-1.5)", "-2"},
		{"math.round(-2.5)", "-3"},
		{"math.round(1.4)", "1"},
		{"math.round(-1.4)", "-1"},
		{"math.round(0.49999999999999994)", "0"},
		{"type(math.round(1.5))", "\"INTEGER\""},
		{"type(math.floor(1.5))", "\"INTEGER\""},
		{"math.min(1, 2)", "1"},
		{"math.min(2, 1, 3)", "1"},
		{"math.min(1.5, 2)", "1.5"},
		{"math.min(2, 1.5)", "1.5"},
		{"math.max(1, 2)", "2"},
		{"math.max(2, 1, 3)", "3"},
		{"math.max(1, 2.5)", "2.5"},
		{"math.max(-1, -2.5, -0.5)", "-0.5"},
		{"type(math.max(1, 2))", "\"INTEGER\""},
		{"math.min(1, 1.0)", "1"},
		{"math.pi", "3.141592653589793"},
		{"math.tau", "6.283185307179586"},
		{"math.tau == 2 * math.pi", "true"},
		{"type(math.sin)", "\"BUILTIN\""},
		{"str(math.sqrt)", "\"builtin math.sqrt\""},
		{"type(math)", "\"HASH\""},
		{"math[\"pi\"] == math.pi", "true"},
		{"len(keys(math))", "19"},
		{"let f = math.sqrt; f(9)", "3.0"},
		// A variable called math hides the library.
		{"let math = {\"pi\": 3}; math.pi", "3"},
		{"function f(math) { math + 1 } f(1)", "2"},
	})
}

func TestMathErrors(t *testing.T) {
	ResetMath()
	checkErrors(t, []expectation{
		{"math.sqrt(-1)", "[line 1, column 10] runtime error: math.sqrt: can't take the square root of a negative number: -1"},
		{"math.sqrt(-0.5)", "[line 1, column 10] runtime error: math.sqrt: can't take the square root of a negative number: -0.5"},
		{"math.sqrt()", "[line 1, column 10] runtime error: math.sqrt: wrong number of arguments: want 1, got 0"},
		{"math.sqrt(1, 2)", "[line 1, column 10] runtime error: math.sqrt: wrong number of arguments: want 1, got 2"},
		{"math.sqrt(\"4\")", "[line 1, column 10] runtime error: math.sqrt: arguments must be INTEGER or FLOAT, got STRING"},
		{"math.sin(true)", "[line 1, column 9] runtime error: math.sin: arguments must be INTEGER or FLOAT, got BOOLEAN"},
		{"math.atan2(1)", "[line 1, column 11] runtime error: math.atan2: wrong number of arguments: want 2, got 1"},
		{"math.abs([])", "[line 1, column 9] runtime error: math.abs: arguments must be INTEGER or FLOAT, got ARRAY"},
		{"math.floor(\"a\")", "[line 1, column 11] runtime error: math.floor: arguments must be INTEGER or FLOAT, got STRING"},
		{"math.floor(10.0 ** 30)", "[line 1, column 11] runtime error: math.floor: the result is too big for an integer: 1e+30"},
		{"math.round(10.0 ** 19)", "[line 1, column 11] runtime error: math.round: the result is too big for an integer: 10000000000000000000.0"},
		{"math.min(1)", "[line 1, column 9] runtime error: math.min: wrong number of arguments: want at least 2, got 1"},
		{"math.max()", "[line 1, column 9] runtime error: math.max: wrong number of arguments: want at least 2, got 0"},
		{"math.max(1, \"a\")", "[line 1, column 9] runtime error: math.max: arguments must be INTEGER or FLOAT, got STRING"},
		{"math.random(1)", "[line 1, column 12] runtime error: math.random: wrong number of arguments: want 0 or 2, got 1"},
		{"math.random(1, 2, 3)", "[line 1, column 12] runtime error: math.random: wrong number of arguments: want 0 or 2, got 3"},
		{"math.random(1.0, 2)", "[line 1, column 12] runtime error: math.random: arguments must be INTEGER, got FLOAT and INTEGER"},
		{"math.random(5, 1)", "[line 1, column 12] runtime error: math.random: the range is empty: 1 is less than 5"},
		{"math.seed()", "[line 1, column 10] runtime error: math.seed: wrong number of arguments: want 1, got 0"},
		{"math.seed(1.5)", "[line 1, column 10] runtime error: math.seed: argument must be INTEGER, got FLOAT"},
		{"math.nothing(1)", "[line 1, column 13] runtime error: not a function: NULL"},
		{"math.pi()", "[line 1, column 8] runtime error: not a function: FLOAT"},
		{"let math = 1; math.sin(1)", "[line 1, column 19] runtime error: index operator not supported: INTEGER"},
	})
}

func TestMathRandom(t *testing.T) {
	// The same program gives the same numbers after a reset, and different
	// seeds give different ones.
	program := "[math.random(), math.random(), math.random(1, 100), math.random(1, 100)]"
	ResetMath()
	first := run(t, program).Inspect()
	ResetMath()
	if again := run(t, program).Inspect(); again != first {
		t.Errorf("not deterministic: %s then %s", first, again)
	}
	if next := run(t, program).Inspect(); next == first {
		t.Errorf("the sequence did not move on: %s twice", first)
	}

	seeded := "math.seed(42); " + program
	ResetMath()
	a := run(t, seeded).Inspect()
	if run(t, "math.seed(7); "+program).Inspect() == a {
		t.Errorf("different seeds gave the same numbers: %s", a)
	}
	if run(t, seeded).Inspect() != a {
		t.Errorf("the same seed gave different numbers")
	}
	if first == a {
		t.Errorf("seed 42 gave the same numbers as the default seed")
	}

	// The default seed is fixed: this guards against it changing by accident.
	ResetMath()
	checkAll(t, []expectation{{"math.random(1, 1000000)", "779411"}})
}

func TestMathRandomRanges(t *testing.T) {
	ResetMath()
	checkAll(t, []expectation{
		{"let ok = true; for (let i = 0; i < 2000; i = i + 1) { let r = math.random(); if (r < 0 || r >= 1) { ok = false } }; ok", "true"},
		{"let ok = true; for (let i = 0; i < 2000; i = i + 1) { let r = math.random(-2, 2); if (r < -2 || r > 2 || type(r) != \"INTEGER\") { ok = false } }; ok", "true"},
		// Both ends are reachable.
		{"let seen = {}; for (let i = 0; i < 500; i = i + 1) { seen[math.random(1, 3)] = true }; len(seen)", "3"},
		{"math.random(5, 5)", "5"},
		{"math.random(-9223372036854775807 - 1, -9223372036854775807 - 1)", "-9223372036854775808"},
		{"type(math.random(-9223372036854775807 - 1, 9223372036854775807))", "\"INTEGER\""},
		{"type(math.random(0, 9223372036854775807))", "\"INTEGER\""},
		{"math.seed(1)", "null"},
	})
}

// TestMathIsPerRun checks that a program that changes math can't affect the
// next run.
func TestMathIsPerRun(t *testing.T) {
	ResetMath()
	checkAll(t, []expectation{{"math.pi = 3; math.pi", "3"}})
	if got := run(t, "math.pi").Inspect(); got != "3" {
		t.Errorf("math.pi was %s, the change should last for the run", got)
	}
	ResetMath()
	if got := run(t, "math.pi").Inspect(); !strings.HasPrefix(got, "3.14") {
		t.Errorf("math.pi was %s after ResetMath", got)
	}
}
