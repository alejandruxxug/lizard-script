package vm

import (
	"github.com/alejandruxxug/lizard-script/engine/evaluator"
	"github.com/alejandruxxug/lizard-script/engine/object"
	"github.com/alejandruxxug/lizard-script/lang/ast"
	"github.com/alejandruxxug/lizard-script/lang/lexer"
	"github.com/alejandruxxug/lizard-script/lang/parser"
	"io"
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
	{"TokenScan", tokenScanBenchmark},
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

// tokenScanBenchmark is the work of an editor that reads code as tokens:
// 22,000 tokens in arrays of types, lines and texts (what lizard.tokens
// gives with columns), cut in statements the way an editor sees them,
// with the arguments of each call. It indexes arrays with int locals,
// reads small tables of constants with a token's type, compares short
// strings and builds small hashes and arrays.
const tokenScanBenchmark = `
let opens = {"(": true, "[": true, "{": true}
let closes = {")": true, "]": true, "}": true}
let carry = {"+": true, "-": true, "*": true, "/": true, "=": true, ",": true, ".": true, "&&": true, "||": true}
let line_types = [["IDENT", "(", "(", "INT", ",", "INT", ")", ",", "INT", ",", "{", "STRING", ":", "COLOR", ",", "STRING", ":", "INT", "}", ")"],
  ["LET", "IDENT", "=", "IDENT", "(", "(", "INT", ",", "INT", ")", ",", "INT", ",", "FLOAT", ")"],
  ["IDENT", "(", "[", "(", "INT", ",", "INT", ")", ",", "(", "INT", ",", "INT", ")", "]", ",", "{", "STRING", ":", "FLOAT", "}", ")"]]
let line_texts = [["circle", "(", "(", "120", ",", "45", ")", ",", "30", ",", "{", "fill", ":", "#ff8800", ",", "stroke", ":", "2", "}", ")"],
  ["let", "s", "=", "petal", "(", "(", "10", ",", "20", ")", ",", "5", ",", "0.5", ")"],
  ["path", "(", "[", "(", "1", ",", "2", ")", ",", "(", "3", ",", "4", ")", "]", ",", "{", "width", ":", "1.5", "}", ")"]]

function tokens(lines) {
  let p = {"ty": [], "ln": [], "tx": [], "n": 0, "pos": 0}
  for (let l = 0; l < lines; l += 1) {
    let types = line_types[l % 3]
    let texts = line_texts[l % 3]
    for (let i = 0; i < len(types); i += 1) {
      push(p.ty, types[i])
      push(p.ln, l + 1)
      push(p.tx, texts[i])
    }
  }
  p.n = len(p.ty)
  p
}

function extent(p, i) {
  let ty = p.ty
  let ln = p.ln
  let n = p.n
  let depth = 0
  for (let j = i; j < n; j += 1) {
    let k = ty[j]
    if (opens[k] == true) {
      depth += 1
    } elif (closes[k] == true) {
      if (depth == 0) { return j }
      depth -= 1
    } elif (k == ";" && depth == 0) {
      return j + 1
    }
    if (depth == 0 && j + 1 < n && ln[j + 1] > ln[j] && carry[k] != true) { return j + 1 }
  }
  n
}

function args_of(p, from, to) {
  let out = []
  let start = from + 1
  let depth = 0
  for (let k = from + 1; k < to; k += 1) {
    let t = p.ty[k]
    if (opens[t] == true) {
      depth += 1
    } elif (closes[t] == true) {
      depth -= 1
    } elif (t == "," && depth == 0) {
      push(out, [start, k])
      start = k + 1
    }
  }
  out
}

function statements(p) {
  let out = []
  while (p.pos < p.n) {
    let i = p.pos
    let end = math.max(extent(p, i), i + 1)
    let kind = "other"
    let name = ""
    let t = p.ty[i]
    if (t == "LET") {
      kind = "let"
      name = p.tx[i + 1]
    } elif (t == "IDENT" && i + 1 < end && p.ty[i + 1] == "(") {
      kind = "call"
      name = p.tx[i]
    }
    push(out, {"kind": kind, "name": name, "line": p.ln[i], "end_line": p.ln[end - 1], "args": args_of(p, i + 1, end)})
    p.pos = end
  }
  out
}

let found = statements(tokens(1200))
[len(found), found[1].name, len(found[2].args)]`

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
