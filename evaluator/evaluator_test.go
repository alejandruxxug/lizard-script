package evaluator

import (
	"bytes"
	"lizardscript/lexer"
	"lizardscript/object"
	"lizardscript/parser"
	"runtime/debug"
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
		{"function f(n) { if (n == 0) { return 0 } " + strings.Repeat("1 + (", 100) + "f(n - 1)" + strings.Repeat(")", 100) + " } f(10000)",
			"[line 1, column 543] runtime error: stack overflow: the calls and the code inside them nest too deeply"},
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

// TestCallDepthRecovers checks that a stack overflow doesn't leave the
// depth counters high, so the next program can call functions again.
func TestCallDepthRecovers(t *testing.T) {
	run(t, "function loop(n) { loop(n + 1) } loop(0)")
	run(t, "function f(n) { "+strings.Repeat("-", 100)+"f(n - 1) } f(0)")
	if callDepth != 0 || evalDepth != 0 {
		t.Errorf("depth counters left at %d calls and %d evals", callDepth, evalDepth)
	}
	checkAll(t, []expectation{{"function f(n) { if (n == 0) { return 0 } f(n - 1) } f(5000)", "0"}})
}

// nest wraps inner in depth arrays, or in depth hashes when hash is set.
func nest(inner object.Object, depth int, hash bool) object.Object {
	value := inner
	for i := 0; i < depth; i++ {
		if hash {
			h := object.NewHash()
			h.Set(&object.String{Value: "in"}, value)
			value = h
		} else {
			value = &object.Array{Elements: []object.Object{value}}
		}
	}
	return value
}

// TestDeepNesting checks that showing and comparing deeply nested values
// doesn't depend on Go's stack. The stack is limited to 1 MB during the
// test, where code that calls itself once per level crashes after a few
// thousand levels.
func TestDeepNesting(t *testing.T) {
	const depth = 100000
	defer debug.SetMaxStack(debug.SetMaxStack(1 << 20))

	array := nest(&object.Integer{Value: 1}, depth, false)
	shown := array.Inspect()
	if shown != strings.Repeat("[", depth)+"1"+strings.Repeat("]", depth) {
		t.Errorf("deep array shown wrong, starts with %q", shown[:20])
	}

	hash := nest(&object.String{Value: "x"}, depth, true)
	shown = hash.Inspect()
	if shown != strings.Repeat("{\"in\": ", depth)+"\"x\""+strings.Repeat("}", depth) {
		t.Errorf("deep hash shown wrong, starts with %q", shown[:20])
	}

	same := nest(&object.String{Value: "1"}, depth, false)
	different := nest(&object.Integer{Value: 2}, depth, false)
	if !looseEquals(array, same) {
		t.Error("deep arrays should be ==")
	}
	if strictEquals(array, same) {
		t.Error("deep arrays with 1 and \"1\" inside should not be ===")
	}
	if looseEquals(array, different) {
		t.Error("deep arrays with 1 and 2 inside should not be ==")
	}
	if !strictEquals(hash, nest(&object.String{Value: "x"}, depth, true)) {
		t.Error("deep hashes should be ===")
	}
}

func TestDotAccess(t *testing.T) {
	checkAll(t, []expectation{
		{`let h = {"a": 1, "b": {"c": 2}}; h.a`, "1"},
		{`let h = {"a": 1, "b": {"c": 2}}; h.b.c`, "2"},
		{`let h = {"a": 1}; h.missing`, "null"},
		{`let h = {"a": 1}; h.a = 5; h.a`, "5"},
		{`let h = {}; h.x = 1; h.y = 2; h`, `{"x": 1, "y": 2}`},
		{`let h = {"l": [1, 2]}; h.l[1]`, "2"},
		{`let h = {"f": function(x) { x * 2 }}; h.f(4)`, "8"},
		{`function make() { {"x": 7} } make().x`, "7"},
		{`let h = {"a": {}}; h.a.b = 3; h["a"]["b"]`, "3"},
		{`let a = [1]; a[0]`, "1"},
		{`"say \"hi\""`, `"say "hi""`},
		{`len("a\nb\t\\")`, "5"},
	})
}

func TestDotErrors(t *testing.T) {
	tests := []expectation{
		{`let n = 5; n.x`, "[line 1, column 13] runtime error: index operator not supported: INTEGER"},
		{`let n = 5; n["x"]`, "[line 1, column 13] runtime error: index operator not supported: INTEGER"},
		{`let a = [1]; a.x`, "[line 1, column 15] runtime error: array index must be INTEGER, got STRING"},
		{`let n = 5; n.x = 1`, "[line 1, column 13] runtime error: index assignment not supported: INTEGER"},
		{`let s = "ab"; s.x = 1`, "[line 1, column 16] runtime error: strings can't be changed"},
	}
	for _, tt := range tests {
		result := run(t, tt.input)
		err, ok := result.(*object.Error)
		if !ok {
			t.Errorf("input %q - expected an error, got %s", tt.input, result.Inspect())
			continue
		}
		if err.Message != tt.expected {
			t.Errorf("input %q\nexpected: %s\n     got: %s", tt.input, tt.expected, err.Message)
		}
	}
}

// withMaxSteps runs fn with the step limit set to max, and puts it back.
func withMaxSteps(max int, fn func()) {
	MaxSteps = max
	ResetSteps()
	defer func() {
		MaxSteps = 0
		ResetSteps()
	}()
	fn()
}

func TestStepLimit(t *testing.T) {
	tests := []expectation{
		{"while (true) {}", "[line 1, column 1] runtime error: the program ran too long (more than 1000 steps)"},
		{"for (;;) {}", "[line 1, column 1] runtime error: the program ran too long (more than 1000 steps)"},
		{"for (let i = 0; i < 100000; i = i + 1) {}", "[line 1, column 1] runtime error: the program ran too long (more than 1000 steps)"},
		{"function f(n) { f(n + 1) } f(0)", "[line 1, column 18] runtime error: the program ran too long (more than 1000 steps)"},
		{"let i = 0; while (true) { len(\"a\") }", "[line 1, column 12] runtime error: the program ran too long (more than 1000 steps)"},
		{"let i = 0; while (i < 10) { i = i + 1 }; while (true) { i = i + 1 }", "[line 1, column 42] runtime error: the program ran too long (more than 1000 steps)"},
	}
	for _, tt := range tests {
		withMaxSteps(1000, func() {
			result := run(t, tt.input)
			err, ok := result.(*object.Error)
			if !ok {
				t.Errorf("input %q - expected an error, got %s", tt.input, result.Inspect())
				return
			}
			if err.Message != tt.expected {
				t.Errorf("input %q\nexpected: %s\n     got: %s", tt.input, tt.expected, err.Message)
			}
		})
	}
}

func TestStepLimitOff(t *testing.T) {
	// With 0 there is no limit, however many steps the program takes.
	checkAll(t, []expectation{
		{"let i = 0; while (i < 5000) { i = i + 1 }; i", "5000"},
		{"function f(n) { if (n == 0) { 0 } else { f(n - 1) } } f(2000)", "0"},
	})

	// Under the limit it works, and ResetSteps starts the count again.
	withMaxSteps(100, func() {
		for range 3 {
			ResetSteps()
			checkAll(t, []expectation{{"let i = 0; while (i < 50) { i = i + 1 }; i", "50"}})
		}
	})
}

// TestStepLimitRecovers checks that a recursion that never ends is
// stopped by the step limit when it is lower than the call limit, and that
// the depth counters are back to zero afterwards.
func TestStepLimitRecovers(t *testing.T) {
	withMaxSteps(500, func() {
		result := run(t, "function f(n) { f(n + 1) } f(0)")
		if !isError(result) || !strings.Contains(result.Inspect(), "ran too long") {
			t.Errorf("expected the step limit, got %s", result.Inspect())
		}
	})
	if callDepth != 0 || evalDepth != 0 {
		t.Errorf("depth counters left at %d calls and %d evals", callDepth, evalDepth)
	}
}

func TestFloatArithmetic(t *testing.T) {
	checkAll(t, []expectation{
		{"1.5", "1.5"},
		{"10.0", "10.0"},
		{"1.5 + 1", "2.5"},
		{"1 + 1.5", "2.5"},
		{"1.5 + 1.5", "3.0"},
		{"0.1 + 0.2", "0.30000000000000004"},
		{"2 * 0.5", "1.0"},
		{"3 - 0.5", "2.5"},
		{"-1.5", "-1.5"},
		{"--1.5", "1.5"},
		{"-0.0", "-0.0"},
		// Integers with integers stay exact.
		{"1 / 2", "0"},
		{"1 + 2", "3"},
		// / with a float is real division.
		{"1 / 2.0", "0.5"},
		{"1.0 / 2", "0.5"},
		{"-7 / 2.0", "-3.5"},
		{"1 / 3.0", "0.3333333333333333"},
		// // floors, and gives a float.
		{"7.5 // 2", "3.0"},
		{"-7.5 // 2", "-4.0"},
		{"7 // 2.0", "3.0"},
		{"7 // -2.0", "-4.0"},
		{"-7.5 // -2", "3.0"},
		{"0.5 // 2", "0.0"},
		// % pairs with //.
		{"7.5 % 2", "1.5"},
		{"-7.5 % 2", "0.5"},
		{"7.5 % -2", "-0.5"},
		{"-7.5 % -2", "-1.5"},
		{"7 % 2.5", "2.0"},
		{"6.0 % 3", "0.0"},
		{"let a = -7.5; let b = 2.0; (a // b) * b + a % b", "-7.5"},
		{"let a = 7.5; let b = -2; (a // b) * b + a % b", "7.5"},
		// **.
		{"2.0 ** 3", "8.0"},
		{"2 ** 3.0", "8.0"},
		{"2.0 ** -1", "0.5"},
		{"4 ** 0.5", "2.0"},
		{"2.5 ** 2", "6.25"},
		{"0.0 ** 0", "1.0"},
		{"-2.0 ** 2", "-4.0"},
		{"-8.0 ** 0.5", "-2.8284271247461903"},
		{"2 ** 10", "1024"},
		// Precedence is the same as for integers.
		{"1.5 + 2 * 3", "7.5"},
		{"(1.5 + 2) * 2", "7.0"},
	})
}

func TestFloatComparisons(t *testing.T) {
	checkAll(t, []expectation{
		{"1.5 < 2", "true"},
		{"2 < 1.5", "false"},
		{"1.5 > 1", "true"},
		{"1.0 <= 1", "true"},
		{"1 >= 1.5", "false"},
		{"1.5 < 2.5", "true"},
		{"0.1 + 0.2 > 0.3", "true"},
		{"1 == 1.0", "true"},
		{"1.0 == 1", "true"},
		{"1 != 1.0", "false"},
		{"1.5 == 1.5", "true"},
		{"1.5 == 2", "false"},
		{"\"1.5\" == 1.5", "true"},
		{"1.5 == \"1.5\"", "true"},
		{"\"5\" == 5.0", "true"},
		{"\"1.50\" == 1.5", "true"},
		{"\"-2.5\" == -2.5", "true"},
		{"\"abc\" == 1.5", "false"},
		{"\"1e3\" == 1000.0", "false"},
		{"\"1.\" == 1.0", "false"},
		{"\"inf\" == 1.0", "false"},
		{"true == 1.0", "true"},
		{"false == 0.0", "true"},
		{"[] == 0.0", "false"},
		{"\"\" == 0.0", "false"},
		{"[1, 2.0] == [1.0, 2]", "true"},
		{"[1, 2.0] === [1.0, 2]", "false"},
		{"1 === 1.0", "false"},
		{"1.0 === 1.0", "true"},
		{"1.0 !== 1", "true"},
		{"\"1.5\" === 1.5", "false"},
		{"0.0 == -0.0", "true"},
	})
}

func TestFloatTruthinessAndTypes(t *testing.T) {
	checkAll(t, []expectation{
		{"if (0.0) { 1 } else { 2 }", "2"},
		{"if (0.5) { 1 } else { 2 }", "1"},
		{"!0.0", "true"},
		{"!1.5", "false"},
		{"0.0 && true", "false"},
		{"1.5 || false", "true"},
		{"type(1.5)", "\"FLOAT\""},
		{"type(1)", "\"INTEGER\""},
		{"type(1 + 0.5)", "\"FLOAT\""},
		{"type(2 // 1.0)", "\"FLOAT\""},
		{"type(10px)", "\"INTEGER\""},
		{"type(10.5px)", "\"FLOAT\""},
	})
}

func TestFloatPrinting(t *testing.T) {
	checkAll(t, []expectation{
		{"str(1.0)", "\"1.0\""},
		{"str(0.5)", "\"0.5\""},
		{"str(0.1)", "\"0.1\""},
		{"str(-2.25)", "\"-2.25\""},
		{"str(100.0)", "\"100.0\""},
		{"str(123456789.0)", "\"123456789.0\""},
		{"str(10.0 ** 20)", "\"100000000000000000000.0\""},
		{"str(10.0 ** 21)", "\"1e+21\""},
		{"str(10.0 ** 25)", "\"1e+25\""},
		{"str(1.5 * 10.0 ** 22)", "\"1.5e+22\""},
		{"str(0.000001)", "\"0.000001\""},
		{"str(0.0000001)", "\"1e-07\""},
		{"str(0.0)", "\"0.0\""},
		{"str(1 / 3.0)", "\"0.3333333333333333\""},
		{"[1.5, 2.0]", "[1.5, 2.0]"},
		{"{\"a\": 0.5}", "{\"a\": 0.5}"},
	})
}

func TestFloatBuiltins(t *testing.T) {
	checkAll(t, []expectation{
		{"int(1.9)", "1"},
		{"int(-1.9)", "-1"},
		{"int(0.5)", "0"},
		{"int(-0.5)", "0"},
		{"int(3.0)", "3"},
		{"int(7)", "7"},
		{"float(3)", "3.0"},
		{"float(1.5)", "1.5"},
		{"float(\"1.5\")", "1.5"},
		{"float(\"2\")", "2.0"},
		{"float(\"-0.25\")", "-0.25"},
		{"type(float(1))", "\"FLOAT\""},
		{"str(2.0) + \"!\"", "\"2.0!\""},
		{"len(str(1.5))", "3"},
	})
}

func TestFloatErrors(t *testing.T) {
	tests := []expectation{
		{"1.5 / 0", "[line 1, column 5] runtime error: division by zero"},
		{"1.5 / 0.0", "[line 1, column 5] runtime error: division by zero"},
		{"1 / 0.0", "[line 1, column 3] runtime error: division by zero"},
		{"1.5 // 0", "[line 1, column 5] runtime error: division by zero"},
		{"1.5 // 0.0", "[line 1, column 5] runtime error: division by zero"},
		{"1.5 % 0", "[line 1, column 5] runtime error: division by zero"},
		{"1.5 % 0.0", "[line 1, column 5] runtime error: division by zero"},
		{"1 % 0.0", "[line 1, column 3] runtime error: division by zero"},
		{"10.0 ** 308 * 10", "[line 1, column 13] runtime error: the result is not a number"},
		{"10.0 ** 400", "[line 1, column 6] runtime error: the result is not a number"},
		{"-10.0 ** 400", "[line 1, column 7] runtime error: the result is not a number"},
		{"0.0 ** -1", "[line 1, column 5] runtime error: the result is not a number"},
		{"(0.0 - 8.0) ** 0.5", "[line 1, column 13] runtime error: the result is not a number"},
		{"let big = 10.0 ** 308; big + big * 10", "[line 1, column 34] runtime error: the result is not a number"},
		{"let big = 10.0 ** 308; big - big * 10 - big * 10", "[line 1, column 34] runtime error: the result is not a number"},
		{"let tiny = 0.0000001; 10.0 ** 308 / tiny", "[line 1, column 35] runtime error: the result is not a number"},
		{"1.5 + true", "[line 1, column 5] runtime error: type mismatch: FLOAT + BOOLEAN"},
		{"1.5 + \"a\"", "[line 1, column 5] runtime error: type mismatch: FLOAT + STRING"},
		{"\"a\" * 1.5", "[line 1, column 5] runtime error: type mismatch: STRING * FLOAT"},
		{"[1][1.0]", "[line 1, column 4] runtime error: array index must be INTEGER, got FLOAT"},
		{"{1.5: 2}", "[line 1, column 1] runtime error: unusable as hash key: FLOAT"},
		{"{}[1.5]", "[line 1, column 3] runtime error: unusable as hash key: FLOAT"},
		{"let h = {}; h[1.5] = 2", "[line 1, column 14] runtime error: unusable as hash key: FLOAT"},
		{"delete({}, 1.5)", "[line 1, column 7] runtime error: delete: unusable as hash key: FLOAT"},
		{"range(2.0)", "[line 1, column 6] runtime error: range: arguments must be INTEGER, got FLOAT"},
		{"int(\"1.5\")", "[line 1, column 4] runtime error: int: not a whole number: \"1.5\""},
		{"int(10.0 ** 25)", "[line 1, column 4] runtime error: int: the float is too big for an integer: 1e+25"},
		{"float(\"abc\")", "[line 1, column 6] runtime error: float: not a number: \"abc\""},
		{"float(\"1e3\")", "[line 1, column 6] runtime error: float: not a number: \"1e3\""},
		{"float(\"\")", "[line 1, column 6] runtime error: float: not a number: \"\""},
		{"float(true)", "[line 1, column 6] runtime error: float: argument not supported, got BOOLEAN"},
		{"float()", "[line 1, column 6] runtime error: float: wrong number of arguments: want 1, got 0"},
		{"len(1.5)", "[line 1, column 4] runtime error: len: argument not supported, got FLOAT"},
		{"-\"a\"", "[line 1, column 1] runtime error: unknown operator: -STRING"},
	}
	checkErrors(t, tests)
}

func TestUnits(t *testing.T) {
	checkAll(t, []expectation{
		{"180deg", "3.141592653589793"},
		{"45deg", "0.7853981633974483"},
		{"90deg == 0.25turn", "true"},
		{"1turn == 360deg", "true"},
		{"1rad", "1.0"},
		{"1.5rad", "1.5"},
		{"0.25turn", "1.5707963267948966"},
		{"10px", "10"},
		{"10px / 4", "2"},
		{"10.5px", "10.5"},
		{"-45deg", "-0.7853981633974483"},
		{"45deg + 45deg", "1.5707963267948966"},
		{"type(45deg)", "\"FLOAT\""},
		{"type(10px)", "\"INTEGER\""},
		{"180deg == math.pi", "true"},
	})
}

// checkErrors checks that each program fails with exactly the given message.
func checkErrors(t *testing.T, tests []expectation) {
	t.Helper()

	for _, tt := range tests {
		result := run(t, tt.input)
		err, ok := result.(*object.Error)
		if !ok {
			t.Errorf("input %q - expected an error, got %s", tt.input, result.Inspect())
			continue
		}
		if err.Message != tt.expected {
			t.Errorf("input %q\nexpected: %s\n     got: %s", tt.input, tt.expected, err.Message)
		}
	}
}
