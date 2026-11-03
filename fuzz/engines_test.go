package fuzz

import (
	"lizardscript/ast"
	"lizardscript/evaluator"
	"lizardscript/object"
	"lizardscript/token"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// failing reports a difference between the engines on a generated program,
// after shrinking it to the smallest program that still shows one.
func failing(t *testing.T, program *Node, difference string) {
	t.Helper()
	panicked := strings.Contains(difference, "PANIC: ")
	small := Shrink(program, func(input string) bool {
		d := Difference(input)
		return KnownBug(input) == "" && d != "" && strings.Contains(d, "PANIC: ") == panicked
	})
	t.Fatalf("the engines differ.\nminimal program:\n%s\n%s\n\nthe generated program was:\n%s",
		small, Difference(small.String()), program)
}

// FuzzEngines runs generated programs on the evaluator and the VM, which
// must agree on everything: output, value, error, steps and hook calls.
func FuzzEngines(f *testing.F) {
	random := rand.New(rand.NewSource(49))
	for i := 0; i < 40; i++ {
		data := make([]byte, 20+random.Intn(200))
		random.Read(data)
		f.Add(data)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		program := Generate(data)
		input := program.String()
		if Parse(input) == nil {
			t.Fatalf("the generated program doesn't parse:\n%s\n%s", input, CheckParse(input))
		}
		if KnownBug(input) != "" {
			return
		}
		if difference := Difference(input); difference != "" {
			failing(t, program, difference)
		}
	})
}

// FuzzEnginesText runs mutated programs of the repo on both engines. Most
// mutations don't parse and are skipped; the rest must give the same on both.
func FuzzEnginesText(f *testing.F) {
	for _, program := range Corpus() {
		f.Add(program)
	}
	f.Fuzz(func(t *testing.T, input string) {
		if KnownBug(input) != "" {
			return
		}
		if difference := Difference(input); difference != "" {
			small := ShrinkLines(input, func(s string) bool { return KnownBug(s) == "" && Difference(s) != "" })
			t.Fatalf("the engines differ.\nminimal program:\n%s\n%s", small, Difference(small))
		}
	})
}

// TestGeneratedPrograms runs a few hundred generated programs, so a plain
// go test checks the generator and the engines too.
func TestGeneratedPrograms(t *testing.T) {
	random := rand.New(rand.NewSource(2026))
	for i := 0; i < 300; i++ {
		data := make([]byte, 50+random.Intn(400))
		random.Read(data)
		program := Generate(data)
		input := program.String()
		if Parse(input) == nil {
			t.Fatalf("the generated program doesn't parse:\n%s\n%s", input, CheckParse(input))
		}
		if KnownBug(input) != "" {
			continue
		}
		if difference := Difference(input); difference != "" {
			failing(t, program, difference)
		}
	}
}

// TestCorpusAgrees runs every program of the corpus on both engines.
func TestCorpusAgrees(t *testing.T) {
	for _, input := range Corpus() {
		if KnownBug(input) != "" {
			continue
		}
		if difference := Difference(input); difference != "" {
			t.Errorf("the engines differ on\n%s\n%s", input, difference)
		}
	}
}

// TestKnownBugs runs the minimal programs the fuzzers found, kept in
// testdata/bugs, one file per bug, named by what it shows. Each fails until
// its bug is fixed; then the file stays as a regression test.
func TestKnownBugs(t *testing.T) {
	files, _ := filepath.Glob(filepath.Join("testdata", "bugs", "*.liz"))
	for _, file := range files {
		name := strings.TrimSuffix(filepath.Base(file), ".liz")
		t.Run(name, func(t *testing.T) {
			source, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			input := string(source)
			if problem := CheckParse(input); problem != "" {
				t.Fatalf("parser: %s\ninput: %q", problem, input)
			}
			if difference := Difference(input); difference != "" {
				t.Fatalf("the engines differ on\n%s\n%s", input, difference)
			}
		})
	}
}

func TestGenerateIsDeterministic(t *testing.T) {
	tests := [][]byte{nil, {0}, {1, 2, 3, 4, 5}, []byte("a longer program from these bytes")}
	for _, data := range tests {
		first, second := Generate(data).String(), Generate(data).String()
		if first != second {
			t.Errorf("%q gave two programs:\n%s\n---\n%s", data, first, second)
		}
		if Parse(first) == nil {
			t.Errorf("%q gave a program that doesn't parse:\n%s", data, first)
		}
	}
}

func TestShrink(t *testing.T) {
	tests := []struct {
		program  *Node
		needle   string
		expected string
	}{
		{
			&Node{Kind: Program, Kids: []*Node{st("let a = 1"), st("puts(%s)", ex("(%s + %s)", leaf("2"), leaf("nope"))), st("a")}},
			"nope",
			"puts(nope)",
		},
		{
			&Node{Kind: Program, Kids: []*Node{st("if (true) %s", block(st("let b = 2"), st("puts(\"x\")"))), st("1")}},
			"\"x\"",
			"puts(\"x\")",
		},
		{
			&Node{Kind: Program, Kids: []*Node{{Kind: Layer, Format: "layer a", Kids: []*Node{st("puts(1)"), st("puts(2)")}}}},
			"2",
			"puts(2)",
		},
	}
	for _, tt := range tests {
		got := Shrink(tt.program, func(s string) bool { return strings.Contains(s, tt.needle) }).String()
		if got != tt.expected {
			t.Errorf("shrinking %q: expected %q, got %q", tt.program.String(), tt.expected, got)
		}
	}
}

func TestShrinkLines(t *testing.T) {
	input := "let a = 1\nlet b = 2\nputs(nope)\nlet c = 3"
	got := ShrinkLines(input, func(s string) bool { return strings.Contains(s, "nope") })
	if got != "puts(nope)" {
		t.Errorf("expected %q, got %q", "puts(nope)", got)
	}
}

// TestDifferenceSeesEachPart swaps the VM for engines that are wrong in one
// way each, and checks that Difference sees it.
func TestDifferenceSeesEachPart(t *testing.T) {
	real := VM
	defer func() { VM = real }()
	tests := []struct {
		name   string
		engine Engine
	}{
		{"output", func(p *ast.Program, env *object.Environment) object.Object {
			evaluator.Output.Write([]byte("extra\n"))
			return real(p, env)
		}},
		{"value", func(p *ast.Program, env *object.Environment) object.Object {
			real(p, env)
			return &object.Integer{Value: 99}
		}},
		{"error position", func(p *ast.Program, env *object.Environment) object.Object {
			result := real(p, env)
			if err, ok := result.(*object.Error); ok {
				return &object.Error{Message: err.Message, Line: err.Line, Column: err.Column + 1}
			}
			return result
		}},
		{"steps", func(p *ast.Program, env *object.Environment) object.Object {
			result := real(p, env)
			evaluator.CountStep(token.Token{})
			return result
		}},
		{"hooks", func(p *ast.Program, env *object.Environment) object.Object {
			if evaluator.OnLayer != nil {
				evaluator.OnLayer("ghost", true, token.Token{})
			}
			return real(p, env)
		}},
		{"panic", func(p *ast.Program, env *object.Environment) object.Object {
			panic("boom")
		}},
	}
	for _, tt := range tests {
		VM = tt.engine
		if Difference("puts(1)\n1 / 0") == "" {
			t.Errorf("%s: no difference seen", tt.name)
		}
	}
	VM = real
	if d := Difference("puts(1)\n1 / 0"); d != "" {
		t.Errorf("the real engines differ: %s", d)
	}
}
