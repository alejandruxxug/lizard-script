package vm

import (
	"bytes"
	"lizardscript/evaluator"
	"lizardscript/lexer"
	"lizardscript/object"
	"lizardscript/parser"
	"strings"
	"testing"
)

// TestProfileCounts checks the calls the profiler counts, also when a try
// or an error leaves frames, and that every call it opened was closed.
func TestProfileCounts(t *testing.T) {
	source := `function fib(n) { if (n < 2) { return n } fib(n - 1) + fib(n - 2) }
let twice = function(x) { x * 2 }
function fail(n) { if (n == 0) { nope } else { fail(n - 1) } }
fib(10)
for (let i = 0; i < 3; i += 1) { twice(i); len([i]) }
try { fail(3) } catch (e) { 0 }
list.sort([3, 1, 2], function(a, b) { a < b })
fail(2)`
	p := parser.New(lexer.NewNamed(source, "count.liz"))
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		t.Fatal(p.Errors())
	}

	StartProfile()
	result := Run(program, object.NewEnvironment())
	open := len(profiler.open)
	profile := StopProfile()
	if _, ok := result.(*object.Error); !ok {
		t.Fatalf("expected the last fail to stop the run, got %s", result.Inspect())
	}
	if open != 0 {
		t.Errorf("%d calls were left open", open)
	}

	calls := map[string]ProfileRow{}
	for _, row := range append(profile.Functions, profile.Builtins...) {
		calls[row.Name] = row
		if row.Self > row.Total || row.Self < 0 {
			t.Errorf("%s: self %v, total %v", row.Name, row.Self, row.Total)
		}
	}
	expected := map[string]int{
		"program": 1, "fib": 177, "twice": 3, "fail": 4 + 3, "len": 3, "list.sort": 1,
	}
	for name, count := range expected {
		if calls[name].Calls != count {
			t.Errorf("%s: %d calls, want %d", name, calls[name].Calls, count)
		}
	}
	if row := calls["function@7"]; row.Calls < 2 || row.Source != "count.liz" {
		t.Errorf("the comparison of list.sort: %+v", row)
	}
	if calls["fib"].Line != 1 || calls["fib"].Source != "count.liz" {
		t.Errorf("fib's place: %+v", calls["fib"])
	}

	var out bytes.Buffer
	profile.Write(&out)
	for _, want := range []string{"functions by self time", "functions by total time", "builtins by self time",
		"fib (count.liz:1)", "twice (count.liz:2)", "list.sort"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("the written profile has no %q:\n%s", want, out.String())
		}
	}
}

// TestProfileOff checks that nothing is counted outside StartProfile and
// StopProfile, and that a profile without a file name shows lines.
func TestProfileOff(t *testing.T) {
	if StopProfile().Functions != nil {
		t.Error("a profile that never started has rows")
	}
	Run(parse(t, "function f() { 1 } f()"), object.NewEnvironment())
	StartProfile()
	profile := StopProfile()
	if len(profile.Functions) != 0 {
		t.Errorf("counted while off: %+v", profile.Functions)
	}

	StartProfile()
	Run(parse(t, "function f() { 1 } f()"), object.NewEnvironment())
	var out bytes.Buffer
	StopProfile().Write(&out)
	if !strings.Contains(out.String(), "f (line 1)") {
		t.Errorf("expected f (line 1) in:\n%s", out.String())
	}
}

// TestProfileForeign counts a function of the evaluator called from the VM
// as a lizard function.
func TestProfileForeign(t *testing.T) {
	env := object.NewEnvironment()
	evaluator.Eval(parse(t, "let tree = function(x) { x }"), env)
	StartProfile()
	Run(parse(t, "tree(1); tree(2)"), env)
	profile := StopProfile()
	for _, row := range profile.Functions {
		if row.Name == "function@1" && row.Calls == 2 {
			return
		}
	}
	t.Errorf("the evaluator's function was not counted: %+v", profile.Functions)
}
