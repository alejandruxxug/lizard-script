package evaluator

import (
	"bytes"
	"lizardscript/lexer"
	"lizardscript/object"
	"lizardscript/parser"
	"strings"
	"testing"
)

// run parses and evaluates input, failing the test on a syntax error.
func run(t *testing.T, input string) object.Object {
	t.Helper()

	p := parser.New(lexer.New(input))
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		t.Fatalf("input %q - unexpected syntax errors:\n%s", input, strings.Join(p.Errors(), "\n"))
	}
	return Eval(program, object.NewEnvironment())
}

// expectation is a test case: the program and the Inspect of its value.
// Strings are written with quotes to tell them apart from other values.
type expectation struct {
	input    string
	expected string
}

func checkAll(t *testing.T, tests []expectation) {
	t.Helper()

	for _, tt := range tests {
		result := run(t, tt.input)
		if result == nil {
			t.Errorf("input %q - got nil", tt.input)
			continue
		}
		if isError(result) {
			t.Errorf("input %q - unexpected error: %s", tt.input, result.Inspect())
			continue
		}
		got := result.Inspect()
		if result.Type() == object.STRING_OBJ {
			got = "\"" + got + "\""
		}
		if got != tt.expected {
			t.Errorf("input %q - expected %s, got %s", tt.input, tt.expected, got)
		}
	}
}

func TestIntegerArithmetic(t *testing.T) {
	checkAll(t, []expectation{
		{"5", "5"},
		{"-5", "-5"},
		{"--5", "5"},
		{"1 + 2 * 3", "7"},
		{"(1 + 2) * 3", "9"},
		{"10 - 2 - 3", "5"},
		{"2 ** 10", "1024"},
		{"2 ** 3 ** 2", "512"},
		{"-2 ** 2", "-4"},
		{"5 ** 0", "1"},
		{"0 ** 0", "1"},
		{"7 / 2", "3"},
		{"-7 / 2", "-3"},
		{"7 // 2", "3"},
		{"-7 // 2", "-4"},
		{"7 // -2", "-4"},
		{"-7 // -2", "3"},
		{"7 % 3", "1"},
		{"-7 % 3", "2"},
		{"7 % -3", "-2"},
		{"6 % 3", "0"},
		{"9223372036854775807 + 1", "-9223372036854775808"},
		{"(-9223372036854775807 - 1) // -1", "-9223372036854775808"},
		{"(-9223372036854775807 - 1) % -1", "0"},
	})
}

func TestComparisonsAndLogic(t *testing.T) {
	checkAll(t, []expectation{
		{"1 < 2", "true"},
		{"2 <= 2", "true"},
		{"3 > 4", "false"},
		{"3 >= 4", "false"},
		{"\"a\" < \"b\"", "true"},
		{"\"b\" >= \"ba\"", "false"},
		{"!true", "false"},
		{"!0", "true"},
		{"!\"\"", "true"},
		{"![]", "true"},
		{"!{}", "true"},
		{"![0]", "false"},
		{"true && false", "false"},
		{"true || false", "true"},
		{"1 && \"yes\"", "true"},
		{"0 || \"\"", "false"},
		{"1 < 2 && 2 < 3", "true"},
	})
}

func TestShortCircuit(t *testing.T) {
	checkAll(t, []expectation{
		{"false && missing", "false"},
		{"true || missing", "true"},
		{"let calls = 0; let f = function() { calls = calls + 1; true }; false && f(); true || f(); calls", "0"},
	})
}

func TestEquality(t *testing.T) {
	checkAll(t, []expectation{
		{"1 == 1", "true"},
		{"1 != 1", "false"},
		{"1 == true", "true"},
		{"0 == false", "true"},
		{"2 == true", "false"},
		{"\"5\" == 5", "true"},
		{"\"5\" != 5", "false"},
		{"\"05\" == 5", "true"},
		{"\"five\" == 5", "false"},
		{"\"true\" == true", "false"},
		{"1 === 1", "true"},
		{"1 === true", "false"},
		{"\"5\" === 5", "false"},
		{"\"5\" !== 5", "true"},
		{"\"a\" == \"a\"", "true"},
		{"let n = if (false) { 1 }; n == if (false) { 2 }", "true"},
		{"let n = if (false) { 1 }; n == 0", "false"},
		{"let n = if (false) { 1 }; n == false", "false"},
		{"[1, 2] == [1, 2]", "true"},
		{"[1, 2] == [2, 1]", "false"},
		{"[1, \"2\"] == [1, 2]", "true"},
		{"[1, \"2\"] === [1, 2]", "false"},
		{"{\"a\": 1, \"b\": 2} == {\"b\": 2, \"a\": 1}", "true"},
		{"{\"a\": 1} == {\"a\": 2}", "false"},
		{"{1: 1} == {\"1\": 1}", "false"},
		{"let f = function() {}; f == f", "true"},
		{"function() {} == function() {}", "false"},
		{"len == len", "true"},
		{"[] == {}", "false"},
	})
}

func TestStrings(t *testing.T) {
	checkAll(t, []expectation{
		{"\"Hello\" + \" \" + \"Liz\"", "\"Hello Liz\""},
		{"\"ab\" * 3", "\"ababab\""},
		{"3 * \"ab\"", "\"ababab\""},
		{"\"ab\" * 0", "\"\""},
		{"\"lizard\"[0]", "\"l\""},
		{"\"lizard\"[5]", "\"d\""},
		{"\"lizard\"[6]", "null"},
		{"\"lizard\"[-1]", "null"},
		{"\"ñandú\"[4]", "\"ú\""},
		{"len(\"ñandú\")", "5"},
	})
}

func TestLetAndAssign(t *testing.T) {
	checkAll(t, []expectation{
		{"let a = 5; a", "5"},
		{"let a = 5; let b = a * 2; b", "10"},
		{"let a = 5; a = 6; a", "6"},
		{"let a = 1; let b = 2; a = b = 3; a + b", "6"},
		{"let a = 1; let a = 2; a", "2"},
		{"let a = 5", "null"},
		{"let x = 1; if (true) { let x = 2; } x", "1"},
		{"let x = 1; if (true) { x = 2; } x", "2"},
		{"let list = [1, 2, 3]; list[1] = 20; list", "[1, 20, 3]"},
		{"let h = {}; h[\"a\"] = 1; h[\"a\"] = 2; h", "{\"a\": 2}"},
		{"let h = {\"a\": [1]}; h[\"a\"][0] = 5; h", "{\"a\": [5]}"},
		{"let a = [1]; let b = a; b[0] = 9; a", "[9]"},
	})
}

func TestIfExpressions(t *testing.T) {
	checkAll(t, []expectation{
		{"if (true) { 10 }", "10"},
		{"if (false) { 10 }", "null"},
		{"if (1) { 10 } else { 20 }", "10"},
		{"if (0) { 10 } else { 20 }", "20"},
		{"let x = 5; if (x > 10) { 1 } elif (x > 3) { 2 } else { 3 }", "2"},
		{"let x = 1; if (x > 10) { 1 } elif (x > 3) { 2 } else { 3 }", "3"},
		{"let y = if (true) { \"yes\" } else { \"no\" }; y", "\"yes\""},
	})
}

func TestLoops(t *testing.T) {
	checkAll(t, []expectation{
		{"let i = 0; while (i < 5) { i = i + 1; } i", "5"},
		{"let i = 0; while (true) { i = i + 1; if (i == 3) { break; } } i", "3"},
		{"let s = 0; for (let i = 1; i <= 10; i = i + 1) { s = s + i; } s", "55"},
		{"let s = 0; for (let i = 0; ; i = i + 1) { if (i > 4) { break } s = s + i } s", "10"},
		{"let i = 0; for (; i < 3;) { i = i + 1 } i", "3"},
		{"let i = 100; for (let i = 0; i < 3; i = i + 1) {} i", "100"},
		{"while (false) {}", "null"},
		{"let n = 0; for (let i = 0; i < 3; i = i + 1) { for (let j = 0; j < 3; j = j + 1) { if (j == 1) { break } n = n + 1 } } n", "3"},
		{"let fs = []; for (let i = 0; i < 3; i = i + 1) { let j = i; push(fs, function() { j }) } fs[0]() + fs[2]()", "2"},
	})
}

func TestReturn(t *testing.T) {
	checkAll(t, []expectation{
		{"return 10; 9", "10"},
		{"return; 9", "null"},
		{"9; return 2 * 5; 9", "10"},
		{"if (true) { if (true) { return 10; } return 1; }", "10"},
		{"let f = function() { while (true) { return 5 } }; f()", "5"},
		{"let f = function() { for (;;) { if (true) { return 7 } } }; f()", "7"},
		{"let f = function() { return }; f()", "null"},
		{"let f = function(x) { if (x) { return 1 } 2 }; f(true) + f(false)", "3"},
	})
}

func TestFunctions(t *testing.T) {
	checkAll(t, []expectation{
		{"let identity = function(x) { x; }; identity(5)", "5"},
		{"let add = function(x, y) { x + y }; add(5, add(1, 2))", "8"},
		{"function add(x, y) { return x + y; } add(2, 3)", "5"},
		{"function(x) { x * 2 }(4)", "8"},
		{"function f() {} f()", "null"},
		{"function fact(n) { if (n <= 1) { return 1 } n * fact(n - 1) } fact(10)", "3628800"},
		{"function fib(n) { if (n < 2) { return n } fib(n - 1) + fib(n - 2) } fib(15)", "610"},
		{"let adder = function(x) { function(y) { x + y } }; adder(2)(3)", "5"},
		{"function counter() { let n = 0; function() { n = n + 1 } } let c = counter(); c(); c(); c()", "3"},
		{"let x = 1; function f() { x } let x = 2; f()", "2"},
		{"let twice = function(f, x) { f(f(x)) }; twice(function(x) { x * 3 }, 2)", "18"},
		{"function f() { 1 } f", "function() { 1 }"},
	})
}

func TestArraysAndHashes(t *testing.T) {
	checkAll(t, []expectation{
		{"[1, 2 * 2, 3 + 3]", "[1, 4, 6]"},
		{"[]", "[]"},
		{"[1, \"a\", true, [2]]", "[1, \"a\", true, [2]]"},
		{"[1, 2, 3][0]", "1"},
		{"[1, 2, 3][2]", "3"},
		{"[1, 2, 3][3]", "null"},
		{"[1, 2, 3][-1]", "null"},
		{"let i = 1; [1, 2, 3][i + 1]", "3"},
		{"[1, 2] + [3]", "[1, 2, 3]"},
		{"{\"name\": \"Liz\", 1: true, false: [5]}", "{\"name\": \"Liz\", 1: true, false: [5]}"},
		{"{\"a\": 1, \"a\": 2}", "{\"a\": 2}"},
		{"{\"a\": 5}[\"a\"]", "5"},
		{"{\"a\": 5}[\"b\"]", "null"},
		{"{1: \"one\"}[1]", "\"one\""},
		{"{1: \"one\"}[\"1\"]", "null"},
		{"{true: 1}[1 < 2]", "1"},
		{"let k = \"x\"; {k: 1}[\"x\"]", "1"},
		{"{}", "{}"},
	})
}

// TestSelfContaining checks arrays and hashes that contain themselves: they
// are shown with [...] or {...}, and comparing them ends.
func TestSelfContaining(t *testing.T) {
	checkAll(t, []expectation{
		{"let a = [1]; push(a, a); a", "[1, [...]]"},
		{"let a = [1]; push(a, a); str(a)", "\"[1, [...]]\""},
		{"let h = {}; h[\"me\"] = h; h", "{\"me\": {...}}"},
		{"let a = []; let h = {\"list\": a}; push(a, h); h", "{\"list\": [{...}]}"},
		{"let a = [1]; [a, a]", "[[1], [1]]"},
		{"let a = [1]; push(a, a); a == a", "true"},
		{"let a = [1]; push(a, a); let b = [1]; push(b, b); a === b", "true"},
		{"let a = [1]; push(a, a); let b = [\"1\"]; push(b, b); a == b", "true"},
		{"let a = [1]; push(a, a); let b = [\"1\"]; push(b, b); a === b", "false"},
		{"let a = [1]; push(a, a); let b = [2]; push(b, b); a == b", "false"},
		{"let a = [1]; push(a, a); a == [1, a]", "true"},
		{"let a = [1]; push(a, a); a == [1, [1, 2]]", "false"},
		{"let h = {}; h[\"me\"] = h; let g = {}; g[\"me\"] = g; h == g", "true"},
		{"let h = {}; h[\"me\"] = h; let g = {}; g[\"me\"] = g; g[\"x\"] = 1; h == g", "false"},
	})
}

func TestBuiltins(t *testing.T) {
	checkAll(t, []expectation{
		{"len(\"\")", "0"},
		{"len(\"four\")", "4"},
		{"len([1, 2, 3])", "3"},
		{"len({\"a\": 1})", "1"},
		{"let a = [1]; push(a, 2); a", "[1, 2]"},
		{"push([], 1)", "[1]"},
		{"let a = [1, 2]; pop(a) + len(a)", "3"},
		{"first([1, 2])", "1"},
		{"first([])", "null"},
		{"last([1, 2])", "2"},
		{"last([])", "null"},
		{"rest([1, 2, 3])", "[2, 3]"},
		{"rest([])", "null"},
		{"let a = [1, 2]; let b = rest(a); b[0] = 9; a", "[1, 2]"},
		{"keys({\"b\": 1, \"a\": 2})", "[\"b\", \"a\"]"},
		{"values({\"b\": 1, \"a\": 2})", "[1, 2]"},
		{"let h = {\"a\": 1, \"b\": 2}; delete(h, \"a\")", "true"},
		{"let h = {\"a\": 1, \"b\": 2}; delete(h, \"a\"); h[\"c\"] = 3; h", "{\"b\": 2, \"c\": 3}"},
		{"delete({}, 1)", "false"},
		{"type(1)", "\"INTEGER\""},
		{"type(\"\")", "\"STRING\""},
		{"type(if (false) {})", "\"NULL\""},
		{"type(len)", "\"BUILTIN\""},
		{"type(function() {})", "\"FUNCTION\""},
		{"str(42) + \"!\"", "\"42!\""},
		{"str([1, \"a\"])", "\"[1, \"a\"]\""},
		{"int(\"42\") + 1", "43"},
		{"int(\"-7\")", "-7"},
		{"int(true)", "1"},
		{"int(5)", "5"},
		{"range(4)", "[0, 1, 2, 3]"},
		{"range(2, 5)", "[2, 3, 4]"},
		{"range(5, 2)", "[]"},
		{"range(-2)", "[]"},
		{"let len = function(x) { 99 }; len(\"a\")", "99"},
	})
}

func TestPuts(t *testing.T) {
	var out bytes.Buffer
	original := Output
	Output = &out
	defer func() { Output = original }()

	result := run(t, "puts(\"Hello\", 1, [\"a\"], true); puts()")
	if result != NULL {
		t.Errorf("puts should return null, got %s", result.Inspect())
	}
	if out.String() != "Hello 1 [\"a\"] true\n\n" {
		t.Errorf("unexpected output %q", out.String())
	}
}

func TestErrors(t *testing.T) {
	tests := []expectation{
		{"5 + true", "[line 1, column 3] runtime error: type mismatch: INTEGER + BOOLEAN"},
		{"5 + true; 5", "[line 1, column 3] runtime error: type mismatch: INTEGER + BOOLEAN"},
		{"-true", "[line 1, column 1] runtime error: unknown operator: -BOOLEAN"},
		{"true + false", "[line 1, column 6] runtime error: unknown operator: BOOLEAN + BOOLEAN"},
		{"\"a\" - \"b\"", "[line 1, column 5] runtime error: unknown operator: STRING - STRING"},
		{"1 < \"2\"", "[line 1, column 3] runtime error: type mismatch: INTEGER < STRING"},
		{"if (10 > 1) {\n  true + false;\n}", "[line 2, column 8] runtime error: unknown operator: BOOLEAN + BOOLEAN"},
		{"foobar", "[line 1, column 1] runtime error: identifier not found: foobar"},
		{"1 / 0", "[line 1, column 3] runtime error: division by zero"},
		{"1 // 0", "[line 1, column 3] runtime error: division by zero"},
		{"1 % 0", "[line 1, column 3] runtime error: division by zero"},
		{"2 ** -1", "[line 1, column 3] runtime error: negative exponent: 2 ** -1"},
		{"\"a\" * -1", "[line 1, column 5] runtime error: can't repeat a string a negative number of times: -1"},
		{"\"ab\" * 9223372036854775807", "[line 1, column 6] runtime error: string too long"},
		{"x = 1", "[line 1, column 1] runtime error: can't assign to x: it was never declared, use let x = ..."},
		{"if (true) { let x = 1 } x = 2", "[line 1, column 25] runtime error: can't assign to x: it was never declared, use let x = ..."},
		{"let a = [1]; a[1] = 2", "[line 1, column 15] runtime error: index out of range: 1, the array has 1 elements"},
		{"let a = [1]; a[\"0\"] = 2", "[line 1, column 15] runtime error: array index must be INTEGER, got STRING"},
		{"let s = \"ab\"; s[0] = \"c\"", "[line 1, column 16] runtime error: strings can't be changed"},
		{"let n = 1; n[0] = 1", "[line 1, column 13] runtime error: index assignment not supported: INTEGER"},
		{"{\"name\": \"Liz\"}[function(x) { x }]", "[line 1, column 16] runtime error: unusable as hash key: FUNCTION"},
		{"{[1]: 2}", "[line 1, column 1] runtime error: unusable as hash key: ARRAY"},
		{"[1][\"a\"]", "[line 1, column 4] runtime error: array index must be INTEGER, got STRING"},
		{"5[0]", "[line 1, column 2] runtime error: index operator not supported: INTEGER"},
		{"5()", "[line 1, column 2] runtime error: not a function: INTEGER"},
		{"function f(x) { x } f()", "[line 1, column 22] runtime error: wrong number of arguments: want 1, got 0"},
		{"function f(x) { x + nope } f(1)", "[line 1, column 21] runtime error: identifier not found: nope"},
		{"len(1)", "[line 1, column 4] runtime error: len: argument not supported, got INTEGER"},
		{"len(\"a\", \"b\")", "[line 1, column 4] runtime error: len: wrong number of arguments: want 1, got 2"},
		{"push(1, 1)", "[line 1, column 5] runtime error: push: argument must be ARRAY, got INTEGER"},
		{"pop([])", "[line 1, column 4] runtime error: pop: the array is empty"},
		{"keys([])", "[line 1, column 5] runtime error: keys: argument must be HASH, got ARRAY"},
		{"int(\"five\")", "[line 1, column 4] runtime error: int: not a whole number: \"five\""},
		{"range(0, 100000000000)", "[line 1, column 6] runtime error: range: range too long: more than 16777216 elements"},
		{"let i = 0; while (i < 3) { i = i + \"1\" }", "[line 1, column 34] runtime error: type mismatch: INTEGER + STRING"},
		{"for (let i = 0; i < 3; i = i + true) {}", "[line 1, column 30] runtime error: type mismatch: INTEGER + BOOLEAN"},
		{"[1, 2 + true, 3]", "[line 1, column 7] runtime error: type mismatch: INTEGER + BOOLEAN"},
		{"function loop(n) { loop(n + 1) } loop(0)", "[line 1, column 24] runtime error: stack overflow: more than 10000 nested calls"},
	}

	for _, tt := range tests {
		result := run(t, tt.input)
		err, ok := result.(*object.Error)
		if !ok {
			t.Errorf("input %q - expected an error, got %s %s", tt.input, result.Type(), result.Inspect())
			continue
		}
		if err.Message != tt.expected {
			t.Errorf("input %q\nexpected: %s\n     got: %s", tt.input, tt.expected, err.Message)
		}
	}
}

// TestCallDepthRecovers checks that a stack overflow doesn't leave the call
// counter high, so the next program can call functions again.
func TestCallDepthRecovers(t *testing.T) {
	run(t, "function loop(n) { loop(n + 1) } loop(0)")
	checkAll(t, []expectation{{"function f(n) { if (n == 0) { return 0 } f(n - 1) } f(5000)", "0"}})
}
