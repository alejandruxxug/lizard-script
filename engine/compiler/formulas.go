package compiler

import (
	"github.com/alejandruxxug/lizard-script/engine/code"
	"github.com/alejandruxxug/lizard-script/lang/ast"
)

// Formulas. Each arithmetic operator makes an object for its result, and in
// x * 1.5 + y / 3.0 the two in the middle are thrown away at once. When an
// expression is only + - * / (and maybe one comparison at the top) on
// values that can be read without running code, it gets an OpFormula, and
// the VM computes it on Go numbers and makes only the object of the result
// (see code.Formula and vm/formulas.go).
//
// The usual instructions of the expression follow the OpFormula, which
// jumps over them when it has the value. When it doesn't (a value is not a
// number, an operator would fail), the VM runs them, so the value and any
// error are exactly theirs. Reading a leaf runs no code of the program and
// changes nothing, so reading them all first and then again is not seen.

var formulaOperators = map[string]bool{"+": true, "-": true, "*": true, "/": true}

var formulaComparisons = map[string]bool{"<": true, ">": true, "<=": true, ">=": true}

// compileFormula compiles node as an OpFormula followed by its usual
// instructions, and reports whether it did: only when it is a formula and
// has a result in the middle to save (two operators or more).
func (c *Compiler) compileFormula(node *ast.InfixExpression) bool {
	if c.inFormula {
		return false
	}
	shape, ok := formulaShape(node, true)
	if !ok || shape.operators < 2 || shape.depth > code.MaxFormulaDepth {
		return false
	}
	formula := code.Formula{Leaves: make([]code.Leaf, 0, shape.leaves)}
	c.formulaSteps(&formula, node)
	c.unit.Formulas = append(c.unit.Formulas, formula)

	end := c.newLabel()
	at := c.emit(code.OpFormula, len(c.unit.Formulas)-1, 0)
	c.point(at, 1, end)
	// The parts of a formula are not formulas of their own.
	c.inFormula = true
	c.compileInfix(node)
	c.inFormula = false
	c.place(end)
	return true
}

// shape is the size of a formula: its leaves, its operators, and the most
// values it has waiting at once while it is computed.
type shape struct {
	leaves, operators, depth int
}

// formulaShape measures node and reports whether it is a formula. A
// comparison is only allowed at the top.
func formulaShape(node ast.Expression, top bool) (shape, bool) {
	if isLeaf(node) {
		return shape{leaves: 1, depth: 1}, true
	}
	infix, isInfix := node.(*ast.InfixExpression)
	if !isInfix || !(formulaOperators[infix.Operator] || top && formulaComparisons[infix.Operator]) {
		return shape{}, false
	}
	left, leftOk := formulaShape(infix.Left, false)
	right, rightOk := formulaShape(infix.Right, false)
	return shape{
		leaves:    left.leaves + right.leaves,
		operators: left.operators + right.operators + 1,
		// The left side's value waits while the right side is computed.
		depth: max(left.depth, right.depth+1),
	}, leftOk && rightOk
}

// isLeaf reports whether node can be read without running code: a name, a
// number literal, or a dot access on one of them.
func isLeaf(node ast.Expression) bool {
	switch node := node.(type) {
	case *ast.Identifier, *ast.IntegerLiteral, *ast.FloatLiteral:
		return true
	case *ast.IndexExpression:
		_, isName := node.Index.(*ast.StringLiteral)
		return isName && isLeaf(node.Left)
	}
	return false
}

// formulaSteps adds the steps of node, a formula, in the order the
// instructions would run them.
func (c *Compiler) formulaSteps(formula *code.Formula, node ast.Expression) {
	if isLeaf(node) {
		leaf, local := c.leaf(node)
		// Leaves has room for every leaf, so it never moves and the
		// local's variable can keep a pointer to its leaf.
		formula.Leaves = append(formula.Leaves, leaf)
		if local != nil {
			local.leaves = append(local.leaves, &formula.Leaves[len(formula.Leaves)-1])
		}
		formula.Steps = append(formula.Steps, code.Step{Leaf: len(formula.Leaves) - 1})
		return
	}
	infix := node.(*ast.InfixExpression)
	c.formulaSteps(formula, infix.Left)
	c.formulaSteps(formula, infix.Right)
	formula.Steps = append(formula.Steps, code.Step{Leaf: -1, Op: infixOpcodes[infix.Operator], Token: c.token(infix.Token)})
}

// leaf is the leaf of node, read like compileExpression would read it. A
// plain local's variable comes with it: if a closure captures it later, the
// leaf must read its cell, like the instructions localOp remembers.
func (c *Compiler) leaf(node ast.Expression) (code.Leaf, *symbol) {
	switch node := node.(type) {
	case *ast.IntegerLiteral:
		return code.Leaf{Kind: code.LeafConstant, Index: c.integer(node.Value)}, nil
	case *ast.FloatLiteral:
		return code.Leaf{Kind: code.LeafConstant, Index: c.float(node.Value)}, nil
	case *ast.IndexExpression:
		leaf, local := c.leaf(node.Left)
		name := node.Index.(*ast.StringLiteral)
		leaf.Fields = append(leaf.Fields, code.FieldRead{Field: c.field(name.Value), Token: c.token(node.Token)})
		return leaf, local
	}
	ident := node.(*ast.Identifier)
	loc := c.resolve(ident.Value)
	switch loc.kind {
	case atLocal:
		if loc.sym.captured {
			return code.Leaf{Kind: code.LeafCell, Index: loc.sym.slot}, nil
		}
		return code.Leaf{Kind: code.LeafLocal, Index: loc.sym.slot}, loc.sym
	case atFree:
		return code.Leaf{Kind: code.LeafFree, Index: loc.index}, nil
	}
	name := code.Name{Name: ident.Value, Token: ident.Token, Checked: loc.checked, Final: loc.final}
	return code.Leaf{Kind: code.LeafName, Index: c.name(name)}, nil
}
