// Package fuzz holds the fuzz tests of lizard-script: the parser must never
// crash or hang and must put every error inside the input, and the
// tree-walking evaluator and the VM must agree on every program. The
// programs come from a generator (generator.go) and from the programs of
// the repo (corpus.go). See README.md for how to run them.
package fuzz

import (
	"bytes"
	"fmt"
	"github.com/alejandruxxug/lizard-script/engine/evaluator"
	"github.com/alejandruxxug/lizard-script/engine/object"
	"github.com/alejandruxxug/lizard-script/engine/vm"
	"github.com/alejandruxxug/lizard-script/lang/ast"
	"github.com/alejandruxxug/lizard-script/lang/lexer"
	"github.com/alejandruxxug/lizard-script/lang/parser"
	"github.com/alejandruxxug/lizard-script/lang/token"
	"os"
	"strings"
)

// MaxSteps is the step limit of every run, the same as the one of
// vm/differential_test.go.
const MaxSteps = 20000

// Outcome is everything a run shows: what it printed, its value or error,
// the steps it took and the hook calls it made. A panic of the engine is
// kept in Panic instead of crashing the fuzzer.
type Outcome struct {
	Output string
	Value  string
	Steps  int
	Hooks  []string
	Panic  string
}

func (o Outcome) String() string {
	panicked := ""
	if o.Panic != "" {
		panicked = "PANIC: " + o.Panic + "\n"
	}
	return fmt.Sprintf("%svalue: %s\nsteps: %d\noutput:\n%s\nhooks:\n%s", panicked, o.Value, o.Steps, o.Output, strings.Join(o.Hooks, "\n"))
}

// Same reports whether two runs show the same thing.
func (o Outcome) Same(other Outcome) bool {
	return o.Output == other.Output && o.Value == other.Value && o.Steps == other.Steps &&
		o.Panic == other.Panic && strings.Join(o.Hooks, "\n") == strings.Join(other.Hooks, "\n")
}

// describe writes a value with its type, and an error with its position
// and whether a try can catch it.
func describe(result object.Object) string {
	switch result := result.(type) {
	case nil:
		return "nil"
	case *object.Error:
		return fmt.Sprintf("ERROR %q line %d column %d uncatchable %t", result.Message, result.Line, result.Column, result.Uncatchable)
	default:
		return string(result.Type()) + " " + result.Inspect()
	}
}

// Engine runs a parsed program in an environment.
type Engine func(*ast.Program, *object.Environment) object.Object

// Evaluator and VM are the two engines that are compared.
var (
	Evaluator Engine = func(program *ast.Program, env *object.Environment) object.Object {
		return evaluator.Eval(program, env)
	}
	VM Engine = vm.Run
)

// RunEngine runs a parsed program on one engine with the step limit and,
// when hooks is true, both hooks recording.
func RunEngine(program *ast.Program, engine Engine, hooks bool) (o Outcome) {
	var out bytes.Buffer
	var events []string
	evaluator.Output = &out
	evaluator.MaxSteps = MaxSteps
	evaluator.ResetSteps()
	evaluator.ResetMath()
	if hooks {
		evaluator.OnStatement = func(node ast.Statement) {
			events = append(events, fmt.Sprintf("stmt %p %s", node, firstLine(node.String())))
		}
		evaluator.OnLayer = func(name string, enter bool, tok token.Token) {
			events = append(events, fmt.Sprintf("layer %s %t %d:%d", name, enter, tok.Line, tok.Column))
		}
	}
	defer func() {
		evaluator.Output = os.Stdout
		evaluator.MaxSteps = 0
		evaluator.OnStatement, evaluator.OnLayer = nil, nil
		if r := recover(); r != nil {
			o = Outcome{Output: out.String(), Value: "panic", Steps: evaluator.Steps(), Hooks: events, Panic: fmt.Sprint(r)}
		}
	}()

	result := engine(program, object.NewEnvironment())
	return Outcome{Output: out.String(), Value: describe(result), Steps: evaluator.Steps(), Hooks: events}
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

// Parse parses input, and gives nil when it has syntax errors.
func Parse(input string) *ast.Program {
	p := parser.New(lexer.New(input))
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		return nil
	}
	return program
}

// Difference runs input on both engines, without and with hooks, and
// describes the first difference, or gives "" when they agree, when
// input doesn't parse, or when it is a known difference. A panic of
// either engine counts as a difference too. It doesn't skip the known
// bugs (see KnownBug): the tests of testdata/bugs use it to show them.
func Difference(input string) string {
	if KnownDifference(input) {
		return ""
	}
	for _, hooks := range []bool{false, true} {
		program := Parse(input)
		if program == nil {
			return ""
		}
		tree := RunEngine(program, Evaluator, hooks)
		compiled := RunEngine(program, VM, hooks)
		if !tree.Same(compiled) || tree.Panic != "" {
			return fmt.Sprintf("hooks %t\n--- evaluator:\n%s\n--- vm:\n%s", hooks, tree, compiled)
		}
	}
	return ""
}

// KnownDifference reports the programs that are meant to differ: the
// evaluator's "nest too deeply" limit protects Go's stack from its own
// recursion, which the VM doesn't have. The generator never nests that
// deep; a mutated corpus program might.
func KnownDifference(input string) bool {
	return strings.Count(input, "(")+strings.Count(input, "{")+strings.Count(input, "[") > 2000
}

// knownBugs are the bugs the fuzzers found that are not fixed yet, each with
// its minimal program in testdata/bugs/<name>.liz. The fuzzers skip the
// inputs that would show them again, so they go on to find new ones. When a
// bug is fixed, its entry goes away and its test stays.
var knownBugs = []struct {
	name    string
	parser  bool // a bug of the lexer or the parser, which the parser fuzzers skip too
	matches func(input string) bool
}{
	// None open: library_hash_order_is_random and
	// continuation_byte_at_line_start_is_column_0 are fixed; their tests stay.
}

// KnownBug gives the name of a known bug that input would show, or "".
func KnownBug(input string) string {
	return knownBug(input, false)
}

// KnownParserBug is KnownBug for the bugs of the lexer and the parser.
func KnownParserBug(input string) string {
	return knownBug(input, true)
}

func knownBug(input string, parserOnly bool) string {
	for _, bug := range knownBugs {
		if (bug.parser || !parserOnly) && bug.matches(input) {
			return bug.name
		}
	}
	return ""
}
