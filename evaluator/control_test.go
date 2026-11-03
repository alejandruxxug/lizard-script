package evaluator

import (
	"lizardscript/object"
	"strings"
	"testing"
)

func TestDefaultParameters(t *testing.T) {
	checkAll(t, []expectation{
		{"function f(a, b = 10) { a + b } f(1)", "11"},
		{"function f(a, b = 10) { a + b } f(1, 2)", "3"},
		{"function f(a = 1, b = 2) { a * 10 + b } f()", "12"},
		{"function f(a = 1, b = 2) { a * 10 + b } f(5)", "52"},
		// A default sees the parameters before it.
		{"function f(a, b = a * 2) { b } f(4)", "8"},
		{"function f(a, b = a * 2) { b } f(4, 1)", "1"},
		{"function f(a = 1, b = a + 1, c = a + b) { c } f()", "3"},
		// It runs at each call, in the scope of the function.
		{"let n = 0; function f(a = n) { a } n = 5; f()", "5"},
		{"let n = 0; function f(a = n) { a } n = 5; f(1)", "1"},
		{"function f(x = [1]) { push(x, 2) } f(); f()", "[1, 2]"},
		{"let make = function() { let k = 7; function(a = k) { a } }; make()()", "7"},
		{"function f(a = function() { 3 }) { a() } f()", "3"},
		{"function f(key, values = null) { values } f(\"x\")", "null"},
		{"function f(a = if (true) { return 4 }) { a } f()", "4"},
		{"let f = function(a, b = 2) { a + b }; f(1)", "3"},
		{"function f(a, b = 2) { a } str(f)", "\"function(a, b = 2) { a }\""},
		// A name that was called default is still a name.
		{"let default = 3; default", "3"},
	})
}

func TestDefaultParameterErrors(t *testing.T) {
	checkErrors(t, []expectation{
		{"function f(a, b = 1) {} f()", "[line 1, column 26] runtime error: wrong number of arguments: want 1 to 2, got 0"},
		{"function f(a, b = 1) {} f(1, 2, 3)", "[line 1, column 26] runtime error: wrong number of arguments: want 1 to 2, got 3"},
		{"function f(a) {} f()", "[line 1, column 19] runtime error: wrong number of arguments: want 1, got 0"},
		{"function f(a = 1) {} f(1, 2)", "[line 1, column 23] runtime error: wrong number of arguments: want 0 to 1, got 2"},
		{"function f(a = nope) {} f()", "[line 1, column 16] runtime error: identifier not found: nope"},
		{"function f(a = 1 + true) {} f()", "[line 1, column 18] runtime error: type mismatch: INTEGER + BOOLEAN"},
		// A default can't use a parameter after it.
		{"function f(a = b, b = 1) { a } f()", "[line 1, column 16] runtime error: identifier not found: b"},
	})
}

func TestDefaultsKeepCallDepth(t *testing.T) {
	if callDepth != 0 {
		t.Fatalf("callDepth should start at 0, got %d", callDepth)
	}
	run(t, "function f(a = nope) { a } f()")
	run(t, "function f(a = 1) { nope } f()")
	if callDepth != 0 {
		t.Errorf("callDepth should come back to 0 after errors, got %d", callDepth)
	}
}

func TestTryCatch(t *testing.T) {
	checkAll(t, []expectation{
		// Nothing fails: the value of the try block.
		{"try { 1 + 1 } catch (e) { 0 }", "2"},
		{"try { 1 + 1 } catch { 0 }", "2"},
		// An error: the value of the catch block.
		{"try { nope } catch (e) { 5 }", "5"},
		{"try { nope } catch { 5 }", "5"},
		{"try { nope } catch (e) { e.message }", "\"identifier not found: nope\""},
		{"try { nope } catch (e) { e.line }", "1"},
		{"try { nope } catch (e) { e.column }", "7"},
		{"try { 1 +\n true } catch (e) { e.message }", "\"type mismatch: INTEGER + BOOLEAN\""},
		{"try { 1 +\n true } catch (e) { e.line }", "1"},
		{"try { 1 / 0 } catch (e) { e.message }", "\"division by zero\""},
		{"try { error(\"boom\") } catch (e) { e.message }", "\"boom\""},
		{"try { error(\"boom\") } catch (e) { keys(e) }", "[\"message\", \"line\", \"column\"]"},
		{"try { len(1) } catch (e) { e.message }", "\"len: argument not supported, got INTEGER\""},
		{"try { math.sqrt(-1) } catch (e) { type(e) }", "\"HASH\""},
		// From inside functions, however deep.
		{"function f() { nope } try { f() } catch (e) { e.message }", "\"identifier not found: nope\""},
		{"function f(n) { if (n == 0) { error(\"deep\") } else { f(n - 1) } } try { f(50) } catch (e) { e.message }", "\"deep\""},
		// Scopes: the body and the catch have their own; the name is local.
		{"let x = 1; try { let x = 2; nope } catch (e) { x }", "1"},
		{"let x = 1; try { x = 2; nope } catch (e) { x }", "2"},
		{"let e = 9; try { nope } catch (e) { 1 }; e", "9"},
		{"try { nope } catch (e) { let e = 3; e }", "3"},
		{"let r = 0; try { nope } catch (e) { r = 4 }; r", "4"},
		// An error in the catch is not caught by the same try, but by an outer one.
		{"try { try { nope } catch (e) { error(\"second\") } } catch (e2) { e2.message }", "\"second\""},
		{"try { try { 1 } catch (e) { 2 } ; nope } catch (e) { 3 }", "3"},
		// A try is a statement: after it, the program goes on.
		{"try { nope } catch (e) { 1 }\n7", "7"},
		{"try { nope } catch (e) { 1 };\n7", "7"},
		{"try { nope }\ncatch (e) { 6 }", "6"},
		// The value of a function is the last statement, a try included.
		{"function f() { try { nope } catch (e) { 8 } } f()", "8"},
		// return, break and continue pass through, in either block.
		{"function f() { try { return 1 } catch (e) { return 2 }; 3 } f()", "1"},
		{"function f() { try { nope } catch (e) { return 2 }; 3 } f()", "2"},
		{"let n = 0; while (true) { try { n = n + 1; break } catch (e) {} }; n", "1"},
		{"let n = 0; while (true) { try { nope } catch (e) { n = n + 1; break } }; n", "1"},
		{"let n = 0; let i = 0; while (i < 4) { i = i + 1; try { if (i == 2) { continue }; n = n + 1 } catch (e) {} }; n", "3"},
		{"let n = 0; for (let i = 0; i < 3; i = i + 1) { try { nope } catch (e) { continue }; n = 9 }; n", "0"},
		// A function that hits a try in a loop.
		{"let c = 0; for (let i = 0; i < 5; i = i + 1) { try { if (i % 2 == 0) { error(\"x\") } } catch (e) { c += 1 } }; c", "3"},
		// try and catch are ordinary names elsewhere.
		{"let try = 1; try + 1", "2"},
		{"let try = 1; try", "1"},
		{"let try = 5; if try { 2 }", "2"},
		{"let try = function(x) { x * 2 }; try(4)", "8"},
		{"let try = 1;\ntry +\n1", "2"},
		{"let catch = 3; catch", "3"},
		{"let catch = 3; try { catch } catch (e) { 0 }", "3"},
		{"let try = 1; try = try + 1; try", "2"},
		{"let h = {\"try\": 1}; h.try", "1"},
	})
}

func TestTryDoesNotCatchHostLimits(t *testing.T) {
	tests := []struct{ input, contains string }{
		{"try { while (true) {} } catch (e) { 1 }", "ran too long"},
		{"try { for (;;) {} } catch { 1 }", "ran too long"},
		{"try { nope } catch (e) { while (true) {} }", "ran too long"},
		{"function f() { try { while (true) {} } catch (e) { 1 } } f()", "ran too long"},
		// The error comes out of a builtin that called a function.
		{"try { list.sort([3, 2, 1], function(a, b) { while (true) {} }) } catch (e) { 1 }", "ran too long"},
	}
	for _, tt := range tests {
		withMaxSteps(1000, func() {
			result := run(t, tt.input)
			err, ok := result.(*object.Error)
			if !ok {
				t.Errorf("input %q - expected an error, got %s", tt.input, result.Inspect())
				return
			}
			if !strings.Contains(err.Message, tt.contains) || !err.Uncatchable {
				t.Errorf("input %q - expected an uncatchable error with %q, got %v %q",
					tt.input, tt.contains, err.Uncatchable, err.Message)
			}
		})
	}

	// Stack overflow, too.
	result := run(t, "function f(n) { f(n + 1) } try { f(0) } catch (e) { 1 }")
	err, ok := result.(*object.Error)
	if !ok || !strings.Contains(err.Message, "stack overflow") || !err.Uncatchable {
		t.Errorf("stack overflow should not be caught, got %s", result.Inspect())
	}
	if callDepth != 0 {
		t.Errorf("callDepth should come back to 0, got %d", callDepth)
	}
}

func TestTryErrorsOutsideTheTry(t *testing.T) {
	checkErrors(t, []expectation{
		{"try { nope } catch (e) { e.message + 1 }", "[line 1, column 36] runtime error: type mismatch: STRING + INTEGER"},
		{"try { nope } catch { e }", "[line 1, column 22] runtime error: identifier not found: e"},
		{"try { nope } catch (e) { 1 }; e", "[line 1, column 31] runtime error: identifier not found: e"},
	})
}

func TestContinue(t *testing.T) {
	checkAll(t, []expectation{
		{"let n = 0; let i = 0; while (i < 5) { i = i + 1; if (i == 3) { continue }; n = n + i }; n", "12"},
		{"let n = 0; for (let i = 0; i < 5; i = i + 1) { if (i == 3) { continue }; n = n + i }; n", "7"},
		// In a for loop the update still runs, so this ends.
		{"let n = 0; for (let i = 0; i < 4; i = i + 1) { continue; n = 99 }; n", "0"},
		{"let n = 0; for (let i = 0; i < 4; i += 1) { continue }; n", "0"},
		{"let n = 0; let i = 0; while (i < 3) { i += 1; continue\n n = 99 }; n", "0"},
		// Only the innermost loop.
		{"let n = 0; for (let i = 0; i < 3; i = i + 1) { for (let j = 0; j < 3; j = j + 1) { if (j == 1) { continue }; n = n + 1 } }; n", "6"},
		// Through a layer and a block.
		{"let n = 0; for (let i = 0; i < 3; i = i + 1) {\n  layer a\n    if (i == 1) { continue }\n    n = n + 1\n}\nn", "2"},
		// A function inside a loop has its own rules: continue inside it
		// is a syntax error, tested in the parser. A loop inside works.
		{"function f() { let n = 0; for (let i = 0; i < 3; i = i + 1) { if (i == 0) { continue }; n = n + 1 }; n } f()", "2"},
		// break still works next to it.
		{"let n = 0; while (true) { n += 1; if (n < 3) { continue }; break }; n", "3"},
		// Ordinary names.
		{"let continue = 4; continue + 1", "5"},
		{"let continue = 4; if (true) { continue + 1 }", "5"},
		{"let continue = 4; function f() { continue + 1 } f()", "5"},
		{"let continue = 4; let r = 0; while (r < 1) { r = r + 1; function f() { continue + 1 } f() }; r", "1"},
		{"let continue = 4; let r = 0; while (r < 2) { r = r + 1; continue + 1 }; r", "2"},
		{"let continue = 4; let r = 0; while (r < 2) { r = r + 1; continue = 5 }; continue + 0", "5"},
		{"let continue = function() { 3 }; let r = 0; while (r < 2) { r = r + 1; continue() }; r", "2"},
		{"let continue = 4; let r = 0; while (r < 1) { r = r + 1; continue -\n1 }; r", "1"},
	})
}

func TestCompoundAssignment(t *testing.T) {
	checkAll(t, []expectation{
		{"let x = 1; x += 2; x", "3"},
		{"let x = 5; x -= 2; x", "3"},
		{"let x = 3; x *= 4; x", "12"},
		{"let x = 12; x /= 4; x", "3"},
		{"let x = 7; x /= 2; x", "3"},
		{"let x = 1; x += 2", "3"},
		{"let x = 1.5; x += 1; x", "2.5"},
		{"let s = \"a\"; s += \"b\"; s", "\"ab\""},
		{"let s = \"ab\"; s *= 2; s", "\"abab\""},
		{"let l = [1]; l += [2]; l", "[1, 2]"},
		{"let x = 10; x -= 2 * 3; x", "4"},
		{"let x = 2; x *= 3 + 1; x", "8"},
		{"let x = 1; let y = 1; x += y += 2; x", "4"},
		{"let x = 1; x += 1; x += 1; x", "3"},
		{"let a = [1, 2, 3]; a[1] += 10; a", "[1, 12, 3]"},
		{"let a = [1, 2, 3]; a[0] *= 5; a[2] -= 1; a", "[5, 2, 2]"},
		{"let h = {\"n\": 1}; h.n += 4; h.n", "5"},
		{"let h = {\"n\": 1}; h[\"n\"] -= 4; h.n", "-3"},
		{"let h = {\"a\": {\"b\": 2}}; h.a.b *= 7; h.a.b", "14"},
		{"let a = [1, 2]; a[0] += 1", "2"},
		// The target is evaluated once.
		{"let n = 0; function i() { n += 1; 0 }; let a = [5]; a[i()] += 1; n", "1"},
		{"let n = 0; function i() { n += 1; 0 }; let a = [5]; a[i()] += 1; a[0]", "6"},
		// It reads x before the right side runs, like x = x + e.
		{"let x = 1; function f() { x = 100; 1 }; x += f(); x", "2"},
		// A scope above.
		{"let x = 1; function f() { x += 1 } f(); f(); x", "3"},
		{"let t = 0; for (let i = 1; i <= 4; i += 1) { t += i }; t", "10"},
		// In a for update.
		{"let n = 0; for (let i = 0; i < 6; i += 2) { n += 1 }; n", "3"},
		// Operators next to it keep their meaning.
		{"let x = 2; x = x ** 2; x", "4"},
		{"let x = 7; x = x // 2; x", "3"},
		{"let a = 6; let b = 2; a /b", "3"},
	})
}

func TestCompoundAssignmentErrors(t *testing.T) {
	checkErrors(t, []expectation{
		{"x += 1", "[line 1, column 1] runtime error: identifier not found: x"},
		{"let x = 1; x /= 0", "[line 1, column 14] runtime error: division by zero"},
		{"let x = 1; x += true", "[line 1, column 14] runtime error: type mismatch: INTEGER + BOOLEAN"},
		{"let x = \"a\"; x -= \"b\"", "[line 1, column 16] runtime error: unknown operator: STRING - STRING"},
		{"let x = true; x += true", "[line 1, column 17] runtime error: unknown operator: BOOLEAN + BOOLEAN"},
		{"let a = [1]; a[5] += 1", "[line 1, column 19] runtime error: type mismatch: NULL + INTEGER"},
		{"let h = {}; h.n += 1", "[line 1, column 17] runtime error: type mismatch: NULL + INTEGER"},
		{"let s = \"abc\"; s[0] += \"x\"", "[line 1, column 17] runtime error: strings can't be changed"},
		{"let a = [1]; a[\"k\"] += 1", "[line 1, column 15] runtime error: array index must be INTEGER, got STRING"},
		{"let x = 1; nope += x", "[line 1, column 12] runtime error: identifier not found: nope"},
	})
}

func TestMathInverseFunctions(t *testing.T) {
	ResetMath()
	checkAll(t, []expectation{
		{"math.acos(1)", "0.0"},
		{"math.acos(0) == math.pi / 2", "true"},
		{"math.acos(-1) == math.pi", "true"},
		{"math.asin(0)", "0.0"},
		{"math.asin(1) == math.pi / 2", "true"},
		{"math.asin(-1) == -math.pi / 2", "true"},
		{"math.asin(0.5) * 6 == math.pi", "false"},
		{"math.abs(math.asin(0.5) - 30deg) < 0.000001", "true"},
		{"math.abs(math.acos(0.5) - 60deg) < 0.000001", "true"},
		{"math.abs(math.acos(math.cos(1.0)) - 1.0) < 0.000001", "true"},
		{"type(math.acos(1))", "\"FLOAT\""},
		{"str(math.asin)", "\"builtin math.asin\""},
	})
	checkErrors(t, []expectation{
		{"math.acos(2)", "[line 1, column 10] runtime error: math.acos: can't take the acos of a number outside -1 to 1: 2"},
		{"math.asin(-1.5)", "[line 1, column 10] runtime error: math.asin: can't take the asin of a number outside -1 to 1: -1.5"},
		{"math.asin()", "[line 1, column 10] runtime error: math.asin: wrong number of arguments: want 1, got 0"},
		{"math.acos(\"a\")", "[line 1, column 10] runtime error: math.acos: arguments must be INTEGER or FLOAT, got STRING"},
	})
}
