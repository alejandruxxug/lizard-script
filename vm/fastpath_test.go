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
