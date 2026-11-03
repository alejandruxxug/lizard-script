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
}

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
		compareEngines(t, bm.source, false)
	}
}
