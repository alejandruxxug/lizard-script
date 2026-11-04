package vm

import (
	"fmt"
	"github.com/alejandruxxug/lizard-script/engine/code"
	"github.com/alejandruxxug/lizard-script/engine/compiler"
	"strings"
	"testing"
)

// operandCount is how many operand words op has in quickened code.
func operandCount(op code.Opcode) int {
	switch {
	case op == opSetFree:
		return 1
	case op == opGetLocals, op == opGetLocalConst, op == opLoop:
		return 2
	case op == opIncLocal, op == opDecLocal, op == opIncFree:
		return 3
	case op >= opAddX && op < opAddS:
		return 3
	case op >= opAddS && op < opJumpUnlessLess:
		return 2
	case op >= opJumpUnlessLess && op < opJumpUnlessLessX:
		return 2
	case op >= opJumpUnlessLessX:
		return 4
	}
	def, err := code.Lookup(byte(op))
	if err != nil {
		panic(err)
	}
	return len(def.OperandWidths)
}

// quickText writes quickened code one instruction per line, like
// code.Instructions.String: the word it starts at, its name, its operands.
func quickText(words []uint32) string {
	var out strings.Builder
	for ip := 0; ip < len(words); {
		op := code.Opcode(words[ip])
		n := operandCount(op)
		fmt.Fprintf(&out, "%04d %s", ip, opName(op))
		for _, o := range words[ip+1 : ip+1+n] {
			fmt.Fprintf(&out, " %d", o)
		}
		out.WriteString("\n")
		ip += 1 + n
	}
	return out.String()
}

// quickened compiles input and quickens its first function: the innermost,
// which the compiler finishes first.
func quickened(t *testing.T, input string) string {
	t.Helper()
	unit, err := compiler.Compile(parse(t, input))
	if err != nil {
		t.Fatal(err)
	}
	return quickText(quick(unit.Functions[0]))
}

// TestQuicken checks what the hot shapes of code quicken to, and that the
// instructions they stand for are gone.
func TestQuicken(t *testing.T) {
	c := constantSource
	tests := []struct {
		input  string
		has    []string
		hasNot []string
	}{
		// A for loop: compare and jump on two locals, x += 1 on a local,
		// the jump back counting the step, no OpSkipScope.
		{"function f(n) { let t = 0; for (let i = 0; i < n; i += 1) { t += i % 7 }; t }",
			[]string{"opJumpUnlessLessX 2 0", "opIncLocal 2", fmt.Sprintf("opModX 2 %d", c|1), "opAddSet 1", "opLoop"},
			[]string{"OpSkipScope", "OpJumpNotTruthy", "OpLess", "OpJump "}},
		// An operator on the top value and a constant, then stored.
		{"function f(x) { x = x * 0.5 + 1.5; x }", []string{"opMulX 0", "opAddS", "OpSetLocal 0"}, []string{"OpFormula", "OpAssignLocal"}},
		// Two locals pushed together, a local and a constant.
		{"function f(a, b) { g(a, b) }", []string{"opGetLocals 0 1"}, nil},
		{"function f(a) { g(a, 1) }", []string{"opGetLocalConst 0"}, nil},
		// A comparison of values that are not locals or constants.
		{"function f(a) { if (g(a) < 2) { 1 } }", []string{"opJumpUnlessLess"}, []string{"OpJumpNotTruthy"}},
		// n += 1 on a captured variable, and an assignment to one dropped.
		{"function f() { let n = 0; function() { n += 1; n = 2; n } }", []string{"opIncFree 0", "opSetFree 0"}, []string{"OpAssignFree"}},
		// A constant key needs no check; a computed one does.
		{"function f(v) { {\"a\": v} }", nil, []string{"OpCheckKey"}},
		{"function f(k, v) { {k: v} }", []string{"OpCheckKey"}, nil},
		// A jump into the middle of a run keeps it apart: the end of the
		// if lands on the +, so the 2 before it stays a plain constant.
		{"function f(c, x) { x + (if (c) { 1 } else { 2 }) }", []string{"OpAdd 0"}, []string{"opAddS"}},
	}
	for _, tt := range tests {
		text := quickened(t, tt.input)
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
