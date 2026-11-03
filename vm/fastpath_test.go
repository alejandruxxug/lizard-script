package vm

import (
	"lizardscript/compiler"
	"lizardscript/evaluator"
	"lizardscript/object"
	"testing"
)

// TestFastPathsAgree runs the corners of the VM's fast paths (floats,
// points, dot access, builtin arguments, remembered globals) on both
// engines: each must give the evaluator's value or error.
func TestFastPathsAgree(t *testing.T) {
	for _, input := range fastPaths {
		if !compareEngines(t, input, true) {
			t.Errorf("syntax error in %q", input)
		}
	}
}

var fastPaths = []string{
	// Floats, and floats mixed with integers.
	"[1.5 + 2, 2 - 0.5, 0.5 * 4, 1 / 4.0, 7.0 / 2, 3 * 1.0]",
	"[1.5 < 2, 2 > 2.5, 2.0 <= 2, 3 >= 3.5, 1 < 1.0, 1.0 == 1, 1.0 === 1, 2.5 != 2.5]",
	"[7.5 // 2, 7.5 % 2, -7.5 % 2, 2.0 ** 3, 2 ** 0.5]",
	"1.0 / 0",
	"1 / 0.0",
	"5.5 % 0",
	"(10.0 ** 300) * (10.0 ** 300)",
	"-(10.0 ** 300) - (10.0 ** 300)",
	"0.0 ** -1",
	"1.5 + \"a\"",
	"\"a\" * 2.5",
	"1.5 + true",
	"let x = 0.1; for (let i = 0; i < 10; i += 1) { x = x * 1.1 + 0.2 }; x",
	// Points: each coordinate keeps its type.
	"[(1, 2) + (3, 4), (1, 2) - (0.5, 4), (1, 2, 3) + (1, 1, 1), (1.5, 2.5) - (1.5, 2.5)]",
	"[(1, 2) * 3, (1, 2) * 0.5, 3 * (1, 2), 0.5 * (1, 2, 3), (7, 8) / 2, (7, 8) / 2.0, (1.0, 2) / 4]",
	"(1, 2) / 0",
	"(1.5, 2) / 0.0",
	"0 * (1, 2)",
	"((10.0 ** 300), 1) * (10.0 ** 300)",
	"(1, 2) + (1, 2, 3)",
	"(1, 2) - (1, 2, 3)",
	"(1, 2) * (3, 4)",
	"(1, 2) / (3, 4)",
	"2 / (1, 2)",
	"2 - (1, 2)",
	"(1, 2) + 1",
	"(1, 2) - 1.5",
	"(1, 2) + \"a\"",
	"[(1, 2) == (1, 2), (1, 2) == (1.0, 2.0), (1, 2) === (1.0, 2), (1, 2) != (2, 1), (1, 2) < (3, 4)]",
	"[-(1, 2.5), -(-(1, 2, 3))]",
	"let p = (1, 2) + (0.5, 0.5); [p.x, p.y, type(p.x), type(p.y), p]",
	"let p = (1, 2.5) * 2; [type(p.x), type(p.y), str(p)]",
	"let p = (0.0, 0.0); for (let i = 0; i < 50; i += 1) { p = p + (0.5, 1.25) * 0.5 - (0.1, 0.1) }; p",
	"let a = []; for (let i = 0; i < 3; i += 1) { push(a, (i, i) * 2) }; a",
	// Dot access on every kind of value.
	"let h = {\"a\": 1, \"b\": {\"c\": 2}}; [h.a, h.b.c, h.missing, h[\"a\"], h.b[\"c\"]]",
	"let h = {}; h.a = 5; h.a += 1; [h.a, h]",
	"let p = (1, 2.5); [p.x, p.y]",
	"let p = (1, 2, 3); [p.x, p.y, p.z]",
	"let p = (1, 2); p.z",
	"let p = (1, 2); p.w",
	"let p = (1, 2); p[\"x\"]",
	"let p = (1, 2); p[0]",
	"let c = #102030; [c.r, c.g, c.b, c.a]",
	"let c = #102030; c.x",
	"\"abc\".x",
	"[1, 2].x",
	"null.x",
	"let f = function() { 1 }; f.x",
	"5.x",
	"[math.pi, text.upper(\"a\"), list.copy([1])]",
	"let h = {\"f\": function(x) { x * 2 }}; h.f(21)",
	"let h = {\"x\": 1}; let n = 0; for (let i = 0; i < 20; i += 1) { n += h.x; h.x = i }; [n, h.x]",
	// Dot access fused with the read before it: a local, a local closures
	// share, a name; and a jump that lands between the two.
	"function f(p) { [p.x, p.y] } f((1, 2.5))",
	"function f() { let h = {\"a\": 1}; let r = h.a; let g = function() { h.a + 1 }; h.a = 5; [r, h.a, g()] } f()",
	"function f(a, b, c) { (if (c) { a } else { b }).x } [f((1, 2), (3, 4), true), f((1, 2), (3, 4), false)]",
	"function f(a, b) { let r = []; for (let i = 0; i < 2; i += 1) { push(r, (if (i == 0) { a } else { b }).y) }; r } f((1, 2), (3, 4))",
	"function f() { nope.x } f()",
	"function f() { math.nope } f()",
	"function f() { let math = {\"sin\": 1}; math.sin } f()",
	"nope.x",
	// Small array literals in one block, constant point literals made once.
	"let a = [1, 2]; push(a, 3); let b = [a, [4, 5, 6], []]; push(b[1], 7); b[1][0] = 0; [a, b, len(b[1])]",
	"function f() { [1, 2, 3] } let x = f(); x[0] = 9; [x, f()]",
	"function f() { (0.5, 1) } [f() == f(), f() === f(), f(), f() + (1, 1), type(f().y)]",
	"let r = []; for (let i = 0; i < 3; i += 1) { push(r, (1, 2) + (i, i)) }; r",
	// Builtins get their arguments lent from the stack.
	"let a = []; for (let i = 0; i < 5; i += 1) { push(a, point(i, i + 1)) }; a",
	"let a = [1, 2]; let b = rest(a); push(b, 9); [a, b]",
	"let r = []; for (let i = 0; i < 3; i += 1) { push(r, [math.max(i, 1, 2), math.min(i, 1.5)]) }; r",
	"list.sort([3, 1, 2], function(a, b) { push([], a); a < b })",
	"let s = 0; for (let i = 0; i < 10; i += 1) { s += len(str(i)) + int(1.5) }; s",
	"puts(1, 2.5, (1, 2), #fff)",
	// Remembered globals and builtins: a let or = after a read is seen.
	"function f() { len } let a = f(); let len = 5; [type(a), f()]",
	"function f() { x } let x = 1; let r = [f()]; x = 2; push(r, f()); let x = 3; push(r, f()); r",
	"let r = []; for (let i = 0; i < 3; i += 1) { push(r, g) ; let g = 1 }; r",
	"let n = 0; function bump() { n = n + 1; n } [bump(), bump(), n]",
	"function f() { math } let a = type(f()); let math = 1; [a, f()]",
	"function f() { nope } try { f() } catch (e) { let nope = 4 }; let nope = 7; f()",
	"function f() { later } let r = 0; try { f() } catch (e) { r = e.message }; let later = 1; [r, f()]",
	// Formulas: arithmetic in one instruction, the same values and errors,
	// in the same order, as one instruction per operator.
	"let x = 1.5; let y = 2; [x * 2 + y, x * y - 1 / 4, (x + y) * (x - y) / 2.0, x * 2 < y + 1, y * 3 >= 6, 7 / 2 * 2]",
	"function f(x, y) { [x * 2 + y, x * y - 1 / 4, (x + y) * (x - y) / 2.0, x * 2 < y + 1, y * 3 >= 6, y / 2 * 2] } [f(1.5, 2), f(3, 4)]",
	"function f(a) { [a + 1 - 1, a * 2 / 2, a + 0 > a - 1, a * 1 == a + 0] } f(9223372036854775807)",
	"function f(a, b) { [a + 0 > b + 0, a * 1 >= b * 1, a - b < 0 + 0] } f(9007199254740993, 9007199254740992)",
	"function f(a, b) { [a * 2 < b + 1, a + 0.5 <= b * 1, a - 1 > b - 1.5] } f(1, 1.5)",
	"function f(a, b) { a / b + 1 } f(1, 0)",
	"function f(a, b) { a / b * 2 } f(1.0, 0.0)",
	"function f(a) { a * a - a } f(10.0 ** 300)",
	"function f(a) { a * 10.0 - a * 10.0 } f(10.0 ** 300)",
	"function f(a) { (a - a) / (a - a) + 1 } f(1.5)",
	"function f(s) { s + \"b\" + s } f(\"a\")",
	"function f(s, n) { [s * n + s, s * 1 * n] } f(\"ab\", 2)",
	"function f(s) { s + 1 + 2 } f(\"a\")",
	"function f(p, v) { [p + v * 2 - v, p * 2 + p, v - p / 2] } f((1, 2), (0.5, 0.5))",
	"function f(a, b) { a + b * 1 + a } [f(\"x\", \"y\"), f(1, 2), f(1.5, 2), f((1, 1), (2, 2)), f(1, 2), f(\"x\", \"y\")]",
	"function f(a, p) { a * 2 + p.x } f(null, 5)",
	"function f(p, a) { p.x * 2 + a } f(5, null)",
	"function f(h) { h.x * 2 + h.y.z - h.missing } f({\"x\": 1, \"y\": {\"z\": 2}, \"missing\": 0.5})",
	"function f(h) { h.x * 2 + h.missing } f({\"x\": 1})",
	"function f(p, q) { (p.x - q.x) * (p.x - q.x) + (p.y - q.y) * (p.y - q.y) } f((1, 2), (4.5, 6))",
	"function f(c) { c.r * 2 + c.g - c.a } f(#102030)",
	"function f() { 1.x * 2 + 3 } f()",
	"function f() { math.pi * 2 + math.tau } f()",
	"function f(r) { math.sqrt(r * r - 9 * 1) } f(5)",
	"1 + 2 + 3 + 4 + 5 + 6 + 7 + 8 + 9 + 10 + 11 + 12 + 13 + 14 + 15 + 16 + 17 + 18",
	"function f(a) { a + a + a + a + a + a + a + a + a + a + a + a + a + a + a + a } f(0.5)",
	"function f(a) { [a - (a - (a - (a - (a - (a - (a - a)))))), a / (a - (a * (a - (a + (a - (a * (a - a)))))))] } f(1.5)",
	"function f(a) { a - (a - (a - (a - (a - (a - (a - (a - a))))))) } f(2)",
	"let a = 2; let b = 3; [a * b + a, a * b < a + b]",
	"let a = 2; a * nope + 1",
	"let a = 2; nope * a + 1",
	"function f() { let a = 1.5; let r = a * 2 + a; let g = function() { a }; a = 4.0; [r, a * 2 + a, g()] } f()",
	"function f() { let a = 1.5; let b = 2.0; let g = function() { a * b + 1 }; a = 3.0; [a * b - 1, g()] } f()",
	"let fs = []; for (let i = 0; i < 3; i += 1) { let k = i * 2 + 1; push(fs, function() { k * k + i }) }; [fs[0](), fs[2]()]",
	"function f() { let s = 0; for (let i = 0; i < 10; i += 1) { s += i * 2 + 1 }; s } f()",
	"let s = 1; s += 2 * 3; s -= 1.5 * 2; s *= 2 + 1; s /= 2 - 0.5; s",
	"function f() { let s = 1; s += 2 * 3; s -= 1.5 * 2; s *= 2 + 1; s /= 2 - 0.5; s } f()",
	"function f() { let s = \"a\"; s += \"b\" * 2; s } f()",
	"nope += 1 * 2",
	"function f(h) { h.n += h.n * 2 + 1; h } f({\"n\": 1})",
	"function f() { let g = function() { x * 2 + 1 }; let x = 1; g() } f()",
	"function f() { let g = function() { x * 2 + 1 }; let r = g(); let x = 1; [r, g()] } let x = 10; f()",
	// Dot access remembers what a hash gave until the hash changes.
	"function f(h) { let s = 0; for (let i = 0; i < 6; i += 1) { s += h.x * 2 + 1; if (i == 1) { h.x = 10 }; if (i == 2) { hash.remove(h, \"x\"); h.x = 0.5 }; if (i == 3) { h = hash.copy(h); h.x = 3 } }; s } f({\"x\": 1})",
	"function f(a, b) { let r = []; for (let i = 0; i < 4; i += 1) { let h = if (i % 2 == 0) { a } else { b }; push(r, h.v) }; r } f({\"v\": 1}, {\"v\": 2})",
	"function f(h) { let r = [h.k]; delete(h, \"k\"); push(r, h.k); h.k = 2; push(r, h.k); r } f({\"k\": 1})",
	"function f(h) { let r = [h.k]; h[\"k\"] = 5; push(r, h.k); hash.merge(h, {\"k\": 6}); push(r, h.k); r } f({\"k\": 1})",
	// An assignment of an operator's result is never a signal; one of an
	// if used as a value can be.
	"for (let i = 0; i < 3; i += 1) { let x = 0; x = if (i == 1) { break }; puts(i) }",
	"function f() { let x = 0; x = if (true) { return 5 }; 9 } f()",
	"for (let i = 0; i < 3; i += 1) { let x = 0; x = if (i == 1) { continue }; puts(i) }",
	"let x = 1; for (let i = 0; i < 3; i += 1) { x += 1; x = x * 2 + 1; x = -x; x = [x] }; x",
	// The hash library and same.
	"let h = {\"a\": 1}; let c = hash.copy(h); c.a = 2; [h, c, same(h, c), same(h, h), h == hash.copy(h)]",
	"[hash.keys({\"a\": 1, 2: 3}), hash.values({\"a\": 1}), hash.merge({\"a\": 1}, {\"a\": 2, \"b\": 3}), hash.has({\"a\": null}, \"a\"), hash.remove({\"a\": 1}, \"a\")]",
	"function f() { 1 } let g = f; [same(f, g), same(f, function() { 1 }), same(len, len), same([], []), same(1, 1.0)]",
	"hash.copy(1)",
}

// TestGlobalsFollowTheHost checks that a remembered global read follows
// what a host does between runs of the same compiled program: defining the
// name, and resetting the libraries.
func TestGlobalsFollowTheHost(t *testing.T) {
	compile := func(input string) func(env *object.Environment) string {
		unit, err := compiler.Compile(parse(t, input))
		if err != nil {
			t.Fatal(err)
		}
		return func(env *object.Environment) string { return RunUnit(unit, env).Inspect() }
	}

	evaluator.ResetMath()
	readLen := compile("len")
	env := object.NewEnvironment()
	if got := readLen(env); got != "builtin len" {
		t.Errorf("first read: got %s", got)
	}
	env.Define("len", &object.Integer{Value: 5})
	if got := readLen(env); got != "5" {
		t.Errorf("after the host defined len: got %s", got)
	}
	if got := readLen(object.NewEnvironment()); got != "builtin len" {
		t.Errorf("in another environment: got %s", got)
	}

	readPi := compile("math.pi")
	env = object.NewEnvironment()
	readPi(env)
	Run(parse(t, "math.pi = 3"), env)
	if got := readPi(env); got != "3" {
		t.Errorf("before ResetMath, the same math: got %s", got)
	}
	evaluator.ResetMath()
	if got := readPi(env); got != "3.141592653589793" {
		t.Errorf("after ResetMath, a new math: got %s", got)
	}
}

// TestFieldsFollowTheHost checks that a dot access that remembers what a
// hash gave sees the host's changes to it between runs.
func TestFieldsFollowTheHost(t *testing.T) {
	unit, err := compiler.Compile(parse(t, "function f() { h.x } f()"))
	if err != nil {
		t.Fatal(err)
	}
	key := &object.String{Value: "x"}
	h := object.NewHash()
	h.Set(key, &object.Integer{Value: 1})
	env := object.NewEnvironment()
	env.Define("h", h)

	steps := []struct {
		name     string
		change   func()
		expected string
	}{
		{"first read", func() {}, "1"},
		{"the same hash", func() {}, "1"},
		{"after Set", func() { h.Set(key, &object.Integer{Value: 2}) }, "2"},
		{"after Delete", func() { h.Delete(key) }, "null"},
		{"after Set again", func() { h.Set(key, &object.Integer{Value: 3}) }, "3"},
		{"another hash", func() { other := object.NewHash(); other.Set(key, &object.Integer{Value: 4}); env.Define("h", other) }, "4"},
	}
	for _, step := range steps {
		step.change()
		if got := RunUnit(unit, env).Inspect(); got != step.expected {
			t.Errorf("%s: got %s, expected %s", step.name, got, step.expected)
		}
	}
}
