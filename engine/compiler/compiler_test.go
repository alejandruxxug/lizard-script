package compiler

import (
	"github.com/alejandruxxug/lizard-script/engine/code"
	"github.com/alejandruxxug/lizard-script/lang/lexer"
	"github.com/alejandruxxug/lizard-script/lang/parser"
	"strings"
	"testing"
)

// compile compiles input and returns the disassembly of the program and
// the unit.
func compile(t *testing.T, input string) (string, *code.Unit) {
	t.Helper()
	p := parser.New(lexer.New(input))
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		t.Fatalf("input %q - syntax errors: %v", input, p.Errors())
	}
	unit, err := Compile(program)
	if err != nil {
		t.Fatalf("input %q - %v", input, err)
	}
	return unit.Main.Instructions.String(), unit
}

// Which function of a test's unit to look at.
const (
	program = -1
	inner   = -2 // the innermost function, compiled first
	outer   = -3 // the outermost function, compiled last
)

// TestNameResolution checks what each read of a name compiles to: the
// decision that keeps the evaluator's dynamic scopes.
func TestNameResolution(t *testing.T) {
	tests := []struct {
		input    string
		function int
		has      []string
		hasNot   []string
	}{
		// The program's own scope is the environment, by name.
		{"let x = 1; x", program, []string{"OpDefineGlobal", "OpGetName"}, []string{"OpGetLocal"}},
		// A local after its let is a plain slot.
		{"function f() { let x = 1; x }", outer, []string{"OpSetLocal 0", "OpGetLocal 0"}, []string{"OpGetName"}},
		// A read before the let means the scope outside.
		{"function f() { let y = x; let x = 2 }", outer, []string{"OpGetName"}, []string{"OpGetLocal"}},
		// Parameters are the first slots.
		{"function f(a, b) { b }", outer, []string{"OpGetLocal 1"}, nil},
		// A local a closure uses becomes a cell, also where it was used before.
		{"function f() { let x = 1; x; function() { x } }", outer, []string{"OpSetCell 0", "OpGetCell 0", "OpEnterScope"}, []string{"OpGetLocal"}},
		// The closure reads it as a captured variable, surely set.
		{"function f() { let x = 1; function() { x } }", inner, []string{"OpGetFree 0"}, nil},
		// Made before the let: checked at run time, then the globals.
		{"function f() { let g = function() { y }; let y = 1 }", inner, []string{"OpGetName"}, []string{"OpGetFree"}},
		// A scope with no captured local enters for free.
		{"function f() { if (true) { let y = 1; y } }", outer, []string{"OpGetLocal"}, []string{"OpEnterScope"}},
		// Builtins are found by name after the globals.
		{"function f() { len }", outer, []string{"OpGetName"}, nil},
	}

	for _, tt := range tests {
		text, unit := compile(t, tt.input)
		switch tt.function {
		case outer:
			// Inner functions finish first, so the outermost is last.
			text = unit.Functions[len(unit.Functions)-1].Instructions.String()
		case inner:
			text = unit.Functions[0].Instructions.String()
		}
		for _, want := range tt.has {
			if !strings.Contains(text, want) {
				t.Errorf("input %q - expected %s in:\n%s", tt.input, want, text)
			}
		}
		for _, unwanted := range tt.hasNot {
			if strings.Contains(text, unwanted) {
				t.Errorf("input %q - did not expect %s in:\n%s", tt.input, unwanted, text)
			}
		}
	}
}

// TestNameChains checks the order a read of a name tries things in.
func TestNameChains(t *testing.T) {
	tests := []struct {
		input   string
		checked int
		final   bool
	}{
		// y in the closure: its own function's y (not set yet when made),
		// then the program.
		{"function f() { let g = function() { y }; let y = 1 }", 1, false},
		// Not set yet in the inner function, set in the outer one.
		{"function f() { let y = 1; function h() { let g = function() { y }; let y = 2 } }", 1, true},
		// Nowhere: straight to the globals.
		{"function f() { function() { zz } }", 0, false},
	}
	for _, tt := range tests {
		_, unit := compile(t, tt.input)
		var found *code.Name
		for i := range unit.Names {
			if n := &unit.Names[i]; n.Name == "y" || n.Name == "zz" {
				found = n
			}
		}
		if found == nil {
			t.Fatalf("input %q - no name chain", tt.input)
		}
		if len(found.Checked) != tt.checked || (found.Final >= 0) != tt.final {
			t.Errorf("input %q - chain checked %v final %d, want %d checked, final %t",
				tt.input, found.Checked, found.Final, tt.checked, tt.final)
		}
	}
}

// TestSignals checks how return, break and continue are compiled.
func TestSignals(t *testing.T) {
	tests := []struct {
		input string
		has   []string
	}{
		// A return value as a value of an if: wrapped, not a return.
		{"function f() { let x = if (true) { return 1 } }", []string{"OpMakeReturn"}},
		// A break inside a try closes the try first.
		{"while (true) { try { break } catch {} }", []string{"OpPopHandler\n"}},
		// A break inside a layer ends the layer first.
		{"while (true) {\n  layer a\n    break\n}", []string{"OpLayerExit"}},
		// A statement that could hold a signal value is checked.
		{"function f() { let x = 1; x }", []string{"OpCheckSignal"}},
	}
	for _, tt := range tests {
		_, unit := compile(t, tt.input)
		var text strings.Builder
		text.WriteString(unit.Main.Instructions.String())
		for _, fn := range unit.Functions {
			text.WriteString(fn.Instructions.String())
		}
		for _, want := range tt.has {
			if !strings.Contains(text.String(), want) {
				t.Errorf("input %q - expected %q in:\n%s", tt.input, want, text.String())
			}
		}
	}
}

func TestFunctionShape(t *testing.T) {
	tests := []struct {
		input            string
		params, required int
		atLeastLocals    int
	}{
		{"function(a, b) { let c = 1 }", 2, 2, 3},
		{"function(a, b = 1, c = 2) { a }", 3, 1, 3},
		{"function() {}", 0, 0, 0},
	}
	for _, tt := range tests {
		_, unit := compile(t, tt.input)
		fn := unit.Functions[0]
		if fn.NumParams != tt.params || fn.Required != tt.required || fn.NumLocals < tt.atLeastLocals {
			t.Errorf("input %q - params %d required %d locals %d", tt.input, fn.NumParams, fn.Required, fn.NumLocals)
		}
	}
}

// TestQuickening checks the instructions made for speed: a dot access with
// its name ready, fused with the read before it when nothing jumps between
// them, and lets in a block that leave no null to drop.
func TestQuickening(t *testing.T) {
	tests := []struct {
		input  string
		has    []string
		hasNot []string
	}{
		{"function f(p) { p.x }", []string{"OpGetLocalField 0 0"}, []string{"OpGetField", "OpIndex"}},
		{"function f(p) { p[\"x\"] }", []string{"OpGetLocalField 0 0"}, []string{"OpIndex"}},
		{"function f(p) { p[0] }", []string{"OpIndex"}, []string{"Field"}},
		{"function f() { math.sin }", []string{"OpGetNameField"}, []string{"OpGetField"}},
		{"function f(p) { p.a.b }", []string{"OpGetLocalField 0 0", "OpGetField 1"}, nil},
		{"function f(p) { f(p).x }", []string{"OpCall", "OpGetField"}, []string{"OpGetLocalField"}},
		// A local that a closure shares later becomes a cell, fused too.
		{"function f() { let h = {}; h.a; function() { h } }", []string{"OpGetCellField"}, []string{"OpGetLocalField"}},
		// The end of the if lands between the read and the field.
		{"function f(a, b, c) { (if (c) { a } else { b }).x }", []string{"OpGetField"}, []string{"OpGetLocalField"}},
		// A point of number literals is a constant.
		{"function f() { (0.5, 1) }", []string{"OpConstant"}, []string{"OpPoint", "OpCheckCoord"}},
		{"function f(a) { (a, 1) }", []string{"OpPoint"}, nil},
		// The null of a let in the middle of a block is not made.
		{"function f() { let a = 1; a }", nil, []string{"OpNull"}},
		{"function f() { let a = 1 }", []string{"OpNull"}, nil},
	}
	for _, tt := range tests {
		_, unit := compile(t, tt.input)
		text := unit.Functions[len(unit.Functions)-1].Instructions.String()
		for _, want := range tt.has {
			if !strings.Contains(text, want) {
				t.Errorf("input %q - expected %s in:\n%s", tt.input, want, text)
			}
		}
		for _, unwanted := range tt.hasNot {
			if strings.Contains(text, unwanted) {
				t.Errorf("input %q - did not expect %s in:\n%s", tt.input, unwanted, text)
			}
		}
	}
}

// TestFormulas checks which arithmetic gets an OpFormula: two operators or
// more (or one under a comparison at the top) on names, number literals and
// their dot accesses, and nothing else. The usual instructions follow it,
// with no formula of their own.
func TestFormulas(t *testing.T) {
	tests := []struct {
		input    string
		formulas int
		has      []string
	}{
		{"function f(x, y) { x * 2 + y }", 1, []string{"0010 OpFormula 0 44", "OpMul", "0039 OpAdd 3\n0044 OpReturnValue"}},
		{"function f(p) { p.x * p.x + p.next.y }", 1, []string{"OpGetLocalField", "OpGetField"}},
		{"function f() { math.pi * 2 + 1 }", 1, []string{"OpGetNameField"}},
		{"function f(x) { x * 2 < 3 }", 1, []string{"OpLess"}},
		{"function f(x) { let s = 0; s += x * 2 }", 1, []string{"OpAdd", "OpAssignLocal"}},
		{"function f(x) { (x * 2 + 1) * (x - 1 - 2) }", 1, nil},
		// One operator stays as it was.
		{"function f(x) { x + 1 }", 0, []string{"OpAdd"}},
		{"function f(x) { x < 1 }", 0, []string{"OpLess"}},
		{"function f(x) { let s = 0; s += x }", 0, []string{"OpAdd"}},
		// A comparison inside, other operators, other values: not formulas,
		// though their parts can be.
		{"function f(x) { (x < 2) < 3 + 1 }", 0, nil},
		{"function f(x) { x == 2 * x + 1 }", 1, []string{"OpEqual"}},
		{"function f(x) { x % 2 + 1 }", 0, []string{"OpMod"}},
		{"function f(x) { f(x) * 2 + 1 }", 0, []string{"OpCall"}},
		{"function f(x) { -x * 2 + 1 }", 0, []string{"OpMinus"}},
		{"function f(x) { x * \"a\" + 1 }", 0, nil},
		{"function f(x) { x[0] * 2 + 1 }", 0, []string{"OpIndex"}},
		{"function f(x) { x * 2 * 3 + (1, 2) }", 1, nil},
		{"function f(x) { [x * 2 + 1, x / 2 - 1] }", 2, nil},
		// A long chain waits for one value at a time, one nested to the
		// right waits for all: up to 8 of them, so one more leaves the
		// first operator out.
		{"function f(x) { x+x+x+x+x+x+x+x+x+x+x+x+x+x+x+x+x+x+x+x }", 1, nil},
		{"function f(x) { x-(x-(x-(x-(x-(x-(x-x)))))) }", 1, []string{"0010 OpFormula"}},
		{"function f(x) { x-(x-(x-(x-(x-(x-(x-(x-x))))))) }", 1, []string{"0010 OpGetLocal 0\n0015 OpFormula"}},
		// An assignment of an operator's result needs no signal check.
		{"function f() { let x = 0; x = x * 2 + 1; x = 5 }", 1, nil},
		{"function f() { let x = 0; x = if (true) { 1 } }", 0, []string{"OpCheckSignal"}},
	}
	for _, tt := range tests {
		_, unit := compile(t, tt.input)
		text := unit.Functions[len(unit.Functions)-1].Instructions.String()
		if got := strings.Count(text, "OpFormula"); got != tt.formulas || len(unit.Formulas) != tt.formulas {
			t.Errorf("input %q - %d formulas, expected %d:\n%s", tt.input, got, tt.formulas, text)
		}
		for _, want := range tt.has {
			if !strings.Contains(text, want) {
				t.Errorf("input %q - expected %s in:\n%s", tt.input, want, text)
			}
		}
	}
	_, unit := compile(t, "function f() { let x = 0; x = x * 2 + 1; x = 5 }")
	if text := unit.Functions[0].Instructions.String(); strings.Contains(text, "OpCheckSignal") {
		t.Errorf("an assignment of an operator's result is checked for a signal:\n%s", text)
	}
}

// TestFormulaLeaves checks where each leaf reads from, including a local
// that a closure captures after the formula was compiled.
func TestFormulaLeaves(t *testing.T) {
	tests := []struct {
		input string
		kinds []code.LeafKind
	}{
		{"function f(a) { a * 2 + a }", []code.LeafKind{code.LeafLocal, code.LeafConstant, code.LeafLocal}},
		{"function f() { let a = 1; let r = a * 2 + a; function() { a } }", []code.LeafKind{code.LeafCell, code.LeafConstant, code.LeafCell}},
		{"function f(a) { function() { a * a + 1 } }", []code.LeafKind{code.LeafFree, code.LeafFree, code.LeafConstant}},
		{"a * b + c", []code.LeafKind{code.LeafName, code.LeafName, code.LeafName}},
	}
	for _, tt := range tests {
		_, unit := compile(t, tt.input)
		if len(unit.Formulas) != 1 {
			t.Errorf("input %q - %d formulas, expected 1", tt.input, len(unit.Formulas))
			continue
		}
		leaves := unit.Formulas[0].Leaves
		for i, kind := range tt.kinds {
			if i >= len(leaves) || leaves[i].Kind != kind {
				t.Errorf("input %q - leaves %+v, expected kinds %v", tt.input, leaves, tt.kinds)
				break
			}
		}
	}
}
