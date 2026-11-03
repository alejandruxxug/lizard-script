package vm

import (
	"io"
	"lizardscript/ast"
	"lizardscript/evaluator"
	"lizardscript/lexer"
	"lizardscript/object"
	"lizardscript/parser"
	"os"
	"testing"
)

// The benchmarks run the same programs on both engines:
//
//	go test ./vm -bench . -benchmem
var benchmarks = []struct {
	name, source string
}{
	{"Fib25", `
function fib(n) { if (n < 2) { return n } fib(n - 1) + fib(n - 2) }
fib(25)`},
	{"NestedLoop", `
let total = 0
for (let i = 0; i < 300; i += 1) {
  for (let j = 0; j < 300; j += 1) {
    total = total + i * j % 7
  }
}
total`},
	{"StringBuild", `
let s = ""
for (let i = 0; i < 2000; i += 1) { s += "ab" }
len(s)`},
	{"ArrayPushIndex", `
let a = []
for (let i = 0; i < 20000; i += 1) { push(a, i) }
let sum = 0
for (let i = 0; i < len(a); i += 1) { sum += a[i] }
sum`},
	{"HashGetSet", `
let h = {}
for (let i = 0; i < 5000; i += 1) { h[i % 100] = i }
let sum = 0
for (let i = 0; i < 5000; i += 1) { sum += h[i % 100] }
sum`},
	{"ClosureCalls", `
function counter() { let n = 0; function() { n += 1 } }
let c = counter()
let add = function(a, b) { a + b }
let r = 0
for (let i = 0; i < 20000; i += 1) { c(); r = add(r, i) }
r`},
	// The next ones run in a function, like drawing code does: their
	// variables are locals, not globals looked up by name.
	{"DotAccess", `
function run(p) {
  let sum = 0
  for (let i = 0; i < 20000; i += 1) { sum += p.x + p.y + p.next.z }
  sum
}
run({"x": 1, "y": 2, "next": {"z": 3}})`},
	{"FloatMath", `
function run() {
  let x = 0.5
  let total = 0.0
  for (let i = 0; i < 20000; i += 1) { total = total + x * 1.5 - x / 3.0; x = x + 0.25 }
  total < 1000000000000
}
run()`},
	{"PointMath", `
function run() {
  let p = (0.0, 0.0)
  let v = (0.5, 1.25)
  for (let i = 0; i < 20000; i += 1) { p = p + v * 0.5 - (0.1, 0.1); p = (p.x, p.y) }
  p
}
run()`},
	{"BuiltinLoop", `
function run() {
  let a = []
  let n = 0
  for (let i = 0; i < 20000; i += 1) { push(a, i); n += len(a) % 3 + math.abs(-i) % 2 + math.floor(1.5) }
  n
}
run()`},
	{"TopLevelDots", `
let p = {"x": 1, "y": 2, "next": {"z": 3}}
let sum = 0
for (let i = 0; i < 20000; i += 1) { sum += p.x + p.y + p.next.z }
sum`},
	// Allocation-heavy work of a drawing document: many points and small
	// numbers kept alive, and shapes (hashes) compared field by field.
	{"Build10kPoints", `
function run() {
  let shapes = []
  for (let i = 0; i < 10000; i += 1) {
    let at = (i % 100, i // 100) * 4 + (0.5, 0.5)
    push(shapes, [at, math.floor(at.x), i % 7])
  }
  len(shapes)
}
run()`},
	{"HashEquality", `
function shape(i) { {"kind": "circle", "x": i, "y": 2, "r": 3.5, "fill": #ff0000, "stroke": null, "width": 1, "at": (i, 2), "name": "c"} }
function run() {
  let a = shape(1)
  let b = shape(1)
  let same = 0
  for (let i = 0; i < 5000; i += 1) { if (a === b) { same += 1 } }
  same
}
run()`},
	// Many small hashes, as a document's shapes: build them, copy each one
	// and read fields of every copy (a different hash at each read).
	{"Hashes10k", `
function copy(h) {
  let out = {}
  let names = keys(h)
  for (let i = 0; i < len(names); i += 1) { out[names[i]] = h[names[i]] }
  out
}
function run() {
  let shapes = []
  for (let i = 0; i < 10000; i += 1) { push(shapes, {"kind": "dot", "x": i, "y": i * 2, "r": 1.5}) }
  let copies = []
  for (let i = 0; i < len(shapes); i += 1) { push(copies, copy(shapes[i])) }
  let sum = 0
  for (let i = 0; i < len(copies); i += 1) { let s = copies[i]; sum += s.x + s.y + s.r }
  sum
}
run()`},
	// The same with the hash library's copy.
	{"Hashes10kLib", `
function run() {
  let shapes = []
  for (let i = 0; i < 10000; i += 1) { push(shapes, {"kind": "dot", "x": i, "y": i * 2, "r": 1.5}) }
  let copies = []
  for (let i = 0; i < len(shapes); i += 1) { push(copies, hash.copy(shapes[i])) }
  let sum = 0
  for (let i = 0; i < len(copies); i += 1) { let s = copies[i]; sum += s.x + s.y + s.r }
  sum
}
run()`},
	{"Geometry", geometryBenchmark},
}

// geometryBenchmark is the drawing code of examples/terrarium/clock.liz
// with the window left out: point and float math, dot access and math
// calls in small loops.
const geometryBenchmark = `
let runs = []
let span = function(row, from, to) { push(runs, [row, from, to]) }
function disc(c, r) {
  let rows = math.floor(r)
  for (let dy = -rows; dy <= rows; dy += 1) {
    let half = math.sqrt(r * r - dy * dy)
    span(c.y + dy, c.x - half, c.x + half)
  }
}
function line(a, b, width) {
  let r = width / 2.0
  let top = math.floor(math.min(a.y, b.y) - r)
  let bottom = math.ceil(math.max(a.y, b.y) + r)
  let d = b - a
  let length2 = d.x * d.x + d.y * d.y
  for (let y = top; y <= bottom; y += 1) {
    let left = null
    let right = null
    for (let x = math.floor(math.min(a.x, b.x) - r); x <= math.ceil(math.max(a.x, b.x) + r); x += 1) {
      let p = (x + 0.5, y + 0.5)
      let t = math.max(0, math.min(1, ((p.x - a.x) * d.x + (p.y - a.y) * d.y) / length2))
      if (math.dist(p, a + d * t) <= r) {
        if (left == null) { left = x }
        right = x + 1
      }
    }
    if (left != null) { span(y, left, right) }
  }
}
let center = (160, 160)
function hand_end(turn, length) {
  let angle = turn * math.tau - math.pi / 2
  center + (math.cos(angle) * length, math.sin(angle) * length)
}
disc(center, 150)
for (let mark = 0; mark < 12; mark += 1) { disc(hand_end(mark / 12.0, 132), 4) }
line(center, hand_end(0.3, 75), 9)
line(center, hand_end(0.7, 115), 6)
line(center, hand_end(0.1, 125), 2)
len(runs)`

func parseBenchmark(b *testing.B, source string) *ast.Program {
	p := parser.New(lexer.New(source))
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		b.Fatal(p.Errors())
	}
	return program
}

func runBenchmarks(b *testing.B, engine func(*ast.Program, *object.Environment) object.Object) {
	evaluator.Output = io.Discard
	defer func() { evaluator.Output = os.Stdout }()
	for _, bm := range benchmarks {
		program := parseBenchmark(b, bm.source)
		b.Run(bm.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				if result := engine(program, object.NewEnvironment()); result.Type() == object.ERROR_OBJ {
					b.Fatal(result.Inspect())
				}
			}
		})
	}
}

func BenchmarkTree(b *testing.B) {
	runBenchmarks(b, func(p *ast.Program, env *object.Environment) object.Object { return evaluator.Eval(p, env) })
}

func BenchmarkVM(b *testing.B) {
	runBenchmarks(b, Run)
}

// TestBenchmarksAgree checks that the benchmark programs give the same
// value on both engines, so the numbers compare the same work.
func TestBenchmarksAgree(t *testing.T) {
	for _, bm := range benchmarks {
		if !compareEngines(t, bm.source, false) {
			t.Errorf("%s: syntax error", bm.name)
		}
	}
}
