package compiler

import (
	"lizardscript/ast"
	"lizardscript/code"
	"lizardscript/object"
)

// infixOpcodes are the opcodes of the binary operators other than && and ||.
var infixOpcodes = map[string]code.Opcode{
	"+":   code.OpAdd,
	"-":   code.OpSub,
	"*":   code.OpMul,
	"/":   code.OpDiv,
	"//":  code.OpFloorDiv,
	"%":   code.OpMod,
	"**":  code.OpPow,
	"==":  code.OpEqual,
	"!=":  code.OpNotEqual,
	"===": code.OpExact,
	"!==": code.OpNotExact,
	"<":   code.OpLess,
	">":   code.OpGreater,
	"<=":  code.OpLessEqual,
	">=":  code.OpGreaterEqual,
}

// compileExpression compiles an expression. transparent is whether it is
// in a statement's place (an expression statement), where an if or loop
// lets a return, break or continue go on through it; anywhere else it
// stops them and they become its value.
func (c *Compiler) compileExpression(node ast.Expression, transparent bool) {
	switch node := node.(type) {
	case *ast.IntegerLiteral:
		c.emit(code.OpConstant, c.integer(node.Value))

	case *ast.FloatLiteral:
		c.emit(code.OpConstant, c.float(node.Value))

	case *ast.StringLiteral:
		c.emit(code.OpConstant, c.str(node.Value))

	case *ast.ColorLiteral:
		c.emit(code.OpConstant, c.constant(&object.Color{R: node.R, G: node.G, B: node.B, A: node.A}))

	case *ast.Boolean:
		if node.Value {
			c.emit(code.OpTrue)
		} else {
			c.emit(code.OpFalse)
		}

	case *ast.PointLiteral:
		if point, ok := constantPoint(node); ok {
			// Made once: points can't change, so every run can share it.
			c.emit(code.OpConstant, c.constant(point))
			return
		}
		for _, coord := range node.Coords {
			c.compileExpression(coord, false)
			c.emit(code.OpCheckCoord, c.token(node.Token))
		}
		c.emit(code.OpPoint, len(node.Coords))

	case *ast.Identifier:
		c.read(node.Value, node.Token)

	case *ast.PrefixExpression:
		c.compileExpression(node.Right, false)
		switch node.Operator {
		case "!":
			c.emit(code.OpNot)
		case "-":
			c.emit(code.OpMinus, c.node(node))
		default:
			c.emit(code.OpPrefix, c.node(node))
		}

	case *ast.InfixExpression:
		c.compileInfix(node)

	case *ast.AssignExpression:
		c.compileAssign(node)

	case *ast.IfExpression:
		c.compileIf(node, transparent)

	case *ast.WhileExpression:
		c.compileWhile(node, transparent)

	case *ast.ForExpression:
		c.compileFor(node, transparent)

	case *ast.FunctionLiteral:
		c.compileFunction(node)

	case *ast.CallExpression:
		c.compileExpression(node.Function, false)
		for _, arg := range node.Arguments {
			c.compileExpression(arg, false)
		}
		c.emit(code.OpCall, len(node.Arguments), c.token(node.Token))

	case *ast.ArrayLiteral:
		for _, element := range node.Elements {
			c.compileExpression(element, false)
		}
		c.emit(code.OpArray, len(node.Elements))

	case *ast.IndexExpression:
		c.compileExpression(node.Left, false)
		if name, ok := node.Index.(*ast.StringLiteral); ok {
			c.getField(c.field(name.Value), c.token(node.Token))
			return
		}
		c.compileExpression(node.Index, false)
		c.emit(code.OpIndex, c.token(node.Token))

	case *ast.HashLiteral:
		// Each key is checked before its value runs, as in the evaluator.
		for _, pair := range node.Pairs {
			c.compileExpression(pair.Key, false)
			c.emit(code.OpCheckKey, c.token(node.Token))
			c.compileExpression(pair.Value, false)
		}
		c.emit(code.OpHash, len(node.Pairs))

	default:
		c.emit(code.OpNull)
	}
}

// compileInfix compiles a binary operator. && and || only evaluate their
// right side when the left one doesn't decide, and give a boolean.
func (c *Compiler) compileInfix(node *ast.InfixExpression) {
	c.compileExpression(node.Left, false)

	switch node.Operator {
	case "&&", "||":
		op := code.OpAnd
		if node.Operator == "||" {
			op = code.OpOr
		}
		end := c.newLabel()
		c.jump(op, end)
		c.compileExpression(node.Right, false)
		c.emit(code.OpToBool)
		c.place(end)
		return
	}

	c.compileExpression(node.Right, false)
	c.emitInfix(node.Operator, c.token(node.Token))
}

func (c *Compiler) emitInfix(operator string, tok int) {
	op, ok := infixOpcodes[operator]
	if !ok {
		panic(compileError("unknown operator " + operator))
	}
	c.emit(op, tok)
}

// compileAssign compiles x = e, a[i] = e and their compound forms. For
// x += e, x is read first; for a[i] += e, a and i run once and a[i] is read
// before e. The value is the new value.
func (c *Compiler) compileAssign(node *ast.AssignExpression) {
	switch left := node.Left.(type) {
	case *ast.Identifier:
		if node.Operator != "" {
			c.read(left.Value, left.Token)
		}
		c.compileExpression(node.Value, false)
		if node.Operator != "" {
			c.emitInfix(node.Operator, c.token(node.Token))
		}
		c.assign(left.Value, left.Token)

	case *ast.IndexExpression:
		c.compileExpression(left.Left, false)
		c.compileExpression(left.Index, false)
		if node.Operator != "" {
			c.emit(code.OpIndexKeep, c.token(left.Token))
		}
		c.compileExpression(node.Value, false)
		if node.Operator != "" {
			c.emitInfix(node.Operator, c.token(node.Token))
		}
		c.emit(code.OpSetIndex, c.token(left.Token))

	default:
		c.fail(node.Token, "can't assign to %s", node.Left.String())
	}
}

// getField emits a dot access with a constant name on the value just
// compiled. A local or a name read right before it is fused with it into
// one instruction, the commonest pairs in drawing code (p.x, math.sin).
func (c *Compiler) getField(field, tok int) {
	if at, ok := c.lastIs(code.OpGetLocal); ok {
		c.replaceLast(at, code.OpGetLocalField, operandOf(c.fs.ins, at), field, tok)
		return
	}
	if at, ok := c.lastIs(code.OpGetName); ok {
		c.replaceLast(at, code.OpGetNameField, operandOf(c.fs.ins, at), field, tok)
		return
	}
	c.emit(code.OpGetField, field, tok)
}

// operandOf reads the first operand of the instruction at `at`.
func operandOf(ins []byte, at int) int {
	return int(code.ReadUint32(ins[at+1:]))
}

// constantPoint is the value of a point literal whose coordinates are all
// number literals, like (0.5, 0.5).
func constantPoint(node *ast.PointLiteral) (*object.Point, bool) {
	coords := make([]object.Object, len(node.Coords))
	for i, coord := range node.Coords {
		switch coord := coord.(type) {
		case *ast.IntegerLiteral:
			coords[i] = &object.Integer{Value: coord.Value}
		case *ast.FloatLiteral:
			coords[i] = &object.Float{Value: coord.Value}
		default:
			return nil, false
		}
	}
	return &object.Point{Coords: coords}, true
}
