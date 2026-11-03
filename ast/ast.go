// Package ast defines the nodes of the abstract syntax tree that the parser
// builds. The tree keeps the structure of a program and drops details like
// parentheses and semicolons, so "(1 + 2)" and "1 + 2" produce the same tree.
package ast

import (
	"bytes"
	"lizardscript/token"
	"strings"
)

// Node is implemented by every node of the tree.
type Node interface {
	// TokenLiteral returns the literal of the token the node starts with.
	TokenLiteral() string
	// String turns the node back into code. Operators are wrapped in
	// parentheses to show how they were grouped: "1 + 2 * 3" gives
	// "(1 + (2 * 3))".
	String() string
}

// Statement is a node that performs an action and produces no value.
type Statement interface {
	Node
	statementNode()
}

// Expression is a node that produces a value.
type Expression interface {
	Node
	expressionNode()
}

// Program is the root of the tree: the statements of a source file, in order.
type Program struct {
	Statements []Statement
}

func (p *Program) TokenLiteral() string {
	if len(p.Statements) > 0 {
		return p.Statements[0].TokenLiteral()
	}
	return ""
}

func (p *Program) String() string {
	var out bytes.Buffer
	for _, s := range p.Statements {
		out.WriteString(s.String())
	}
	return out.String()
}

// Statements

// LetStatement declares a variable: let <Name> = <Value>;
// A named function declaration also becomes a LetStatement, so
// function add(x) { ... } is the same as let add = function(x) { ... };
type LetStatement struct {
	Token token.Token // the LET token
	Name  *Identifier
	Value Expression
}

func (ls *LetStatement) statementNode()       {}
func (ls *LetStatement) TokenLiteral() string { return ls.Token.Literal }
func (ls *LetStatement) String() string {
	return ls.TokenLiteral() + " " + ls.Name.String() + " = " + ls.Value.String() + ";"
}

// ReturnStatement leaves a function: return <ReturnValue>;
// ReturnValue is nil in a bare "return;".
type ReturnStatement struct {
	Token       token.Token // the RETURN token
	ReturnValue Expression
}

func (rs *ReturnStatement) statementNode()       {}
func (rs *ReturnStatement) TokenLiteral() string { return rs.Token.Literal }
func (rs *ReturnStatement) String() string {
	if rs.ReturnValue == nil {
		return rs.TokenLiteral() + ";"
	}
	return rs.TokenLiteral() + " " + rs.ReturnValue.String() + ";"
}

// BreakStatement leaves the innermost loop: break;
// The parser only accepts it inside a while or for body.
type BreakStatement struct {
	Token token.Token // the BREAK token
}

func (bs *BreakStatement) statementNode()       {}
func (bs *BreakStatement) TokenLiteral() string { return bs.Token.Literal }
func (bs *BreakStatement) String() string       { return bs.TokenLiteral() + ";" }

// ExpressionStatement lets an expression stand alone as a statement, because
// a program is a list of statements: add(1, 2);
type ExpressionStatement struct {
	Token      token.Token // the first token of the expression
	Expression Expression
}

func (es *ExpressionStatement) statementNode()       {}
func (es *ExpressionStatement) TokenLiteral() string { return es.Token.Literal }
func (es *ExpressionStatement) String() string       { return es.Expression.String() }

// BlockStatement is a list of statements between braces: the body of an if,
// a loop or a function.
type BlockStatement struct {
	Token      token.Token // the '{' token
	Statements []Statement
}

func (bs *BlockStatement) statementNode()       {}
func (bs *BlockStatement) TokenLiteral() string { return bs.Token.Literal }
func (bs *BlockStatement) String() string {
	var out bytes.Buffer
	out.WriteString("{ ")
	for _, s := range bs.Statements {
		out.WriteString(s.String())
		out.WriteString(" ")
	}
	out.WriteString("}")
	return out.String()
}

// Leaf expressions

// Identifier is the name of a variable or function.
type Identifier struct {
	Token token.Token // the IDENT token
	Value string
}

func (i *Identifier) expressionNode()      {}
func (i *Identifier) TokenLiteral() string { return i.Token.Literal }
func (i *Identifier) String() string       { return i.Value }

// IntegerLiteral is a whole number written in the code.
type IntegerLiteral struct {
	Token token.Token // the INT token
	Value int64
}

func (il *IntegerLiteral) expressionNode()      {}
func (il *IntegerLiteral) TokenLiteral() string { return il.Token.Literal }
func (il *IntegerLiteral) String() string       { return il.Token.Literal }

// FloatLiteral is a decimal number written in the code. A number with a unit
// like 45deg is one too, already converted by the lexer.
type FloatLiteral struct {
	Token token.Token // the FLOAT token
	Value float64
}

func (fl *FloatLiteral) expressionNode()      {}
func (fl *FloatLiteral) TokenLiteral() string { return fl.Token.Literal }
func (fl *FloatLiteral) String() string       { return fl.Token.Literal }

// Boolean is true or false.
type Boolean struct {
	Token token.Token // the TRUE or FALSE token
	Value bool
}

func (b *Boolean) expressionNode()      {}
func (b *Boolean) TokenLiteral() string { return b.Token.Literal }
func (b *Boolean) String() string       { return b.Token.Literal }

// StringLiteral is a text between double quotes. Value has no quotes.
type StringLiteral struct {
	Token token.Token // the STRING token
	Value string
}

func (sl *StringLiteral) expressionNode()      {}
func (sl *StringLiteral) TokenLiteral() string { return sl.Token.Literal }
func (sl *StringLiteral) String() string       { return "\"" + sl.Value + "\"" }

// Operators

// PrefixExpression is an operator before its operand: -x, !done
type PrefixExpression struct {
	Token    token.Token // the operator token
	Operator string
	Right    Expression
}

func (pe *PrefixExpression) expressionNode()      {}
func (pe *PrefixExpression) TokenLiteral() string { return pe.Token.Literal }
func (pe *PrefixExpression) String() string {
	return "(" + pe.Operator + pe.Right.String() + ")"
}

// InfixExpression is a binary operator between its operands: a + b, x == y,
// ok && done. The evaluator decides how each operator behaves, including
// whether && and || short-circuit.
type InfixExpression struct {
	Token    token.Token // the operator token
	Left     Expression
	Operator string
	Right    Expression
}

func (ie *InfixExpression) expressionNode()      {}
func (ie *InfixExpression) TokenLiteral() string { return ie.Token.Literal }
func (ie *InfixExpression) String() string {
	return "(" + ie.Left.String() + " " + ie.Operator + " " + ie.Right.String() + ")"
}

// AssignExpression changes something that already exists: x = 1, list[0] = 2
// Left is always an Identifier or an IndexExpression, the parser rejects
// anything else. It is an expression so it can be the update of a for loop.
type AssignExpression struct {
	Token token.Token // the '=' token
	Left  Expression
	Value Expression
}

func (ae *AssignExpression) expressionNode()      {}
func (ae *AssignExpression) TokenLiteral() string { return ae.Token.Literal }
func (ae *AssignExpression) String() string {
	return "(" + ae.Left.String() + " = " + ae.Value.String() + ")"
}

// Control flow

// IfExpression is a conditional. Alternative is nil when there is no else.
// There is no elif node: "if (a) {} elif (b) {}" is built as
// "if (a) {} else { if (b) {} }", with elif as the Token of the inner if.
type IfExpression struct {
	Token       token.Token // the IF or ELIF token
	Condition   Expression
	Consequence *BlockStatement
	Alternative *BlockStatement
}

func (ie *IfExpression) expressionNode()      {}
func (ie *IfExpression) TokenLiteral() string { return ie.Token.Literal }
func (ie *IfExpression) String() string {
	out := "if " + ie.Condition.String() + " " + ie.Consequence.String()
	if ie.Alternative != nil {
		out += " else " + ie.Alternative.String()
	}
	return out
}

// WhileExpression repeats Body while Condition holds.
type WhileExpression struct {
	Token     token.Token // the WHILE token
	Condition Expression
	Body      *BlockStatement
}

func (we *WhileExpression) expressionNode()      {}
func (we *WhileExpression) TokenLiteral() string { return we.Token.Literal }
func (we *WhileExpression) String() string {
	return "while " + we.Condition.String() + " " + we.Body.String()
}

// ForExpression is a C-style loop: for (<Init>; <Condition>; <Update>) <Body>
// Init runs once. Condition is checked before each round and Update runs
// after it. Any of the three can be left out, which leaves it nil.
type ForExpression struct {
	Token     token.Token // the FOR token
	Init      Statement
	Condition Expression
	Update    Expression
	Body      *BlockStatement
}

func (fe *ForExpression) expressionNode()      {}
func (fe *ForExpression) TokenLiteral() string { return fe.Token.Literal }
func (fe *ForExpression) String() string {
	var out bytes.Buffer
	out.WriteString("for (")
	if fe.Init != nil {
		out.WriteString(strings.TrimSuffix(fe.Init.String(), ";"))
	}
	out.WriteString("; ")
	if fe.Condition != nil {
		out.WriteString(fe.Condition.String())
	}
	out.WriteString("; ")
	if fe.Update != nil {
		out.WriteString(fe.Update.String())
	}
	out.WriteString(") ")
	out.WriteString(fe.Body.String())
	return out.String()
}

// Functions

// FunctionLiteral defines a function: function(a, b) { return a + b; }
// Functions are values, so they have no name of their own: a name comes from
// the LetStatement that stores them.
type FunctionLiteral struct {
	Token      token.Token // the FUNCTION token
	Parameters []*Identifier
	Body       *BlockStatement
}

func (fl *FunctionLiteral) expressionNode()      {}
func (fl *FunctionLiteral) TokenLiteral() string { return fl.Token.Literal }
func (fl *FunctionLiteral) String() string {
	params := []string{}
	for _, p := range fl.Parameters {
		params = append(params, p.String())
	}
	return fl.TokenLiteral() + "(" + strings.Join(params, ", ") + ") " + fl.Body.String()
}

// CallExpression calls a function: add(1, 2 * 3)
// Function is any expression that produces a function.
type CallExpression struct {
	Token     token.Token // the '(' token
	Function  Expression
	Arguments []Expression
}

func (ce *CallExpression) expressionNode()      {}
func (ce *CallExpression) TokenLiteral() string { return ce.Token.Literal }
func (ce *CallExpression) String() string {
	return ce.Function.String() + "(" + joinExpressions(ce.Arguments) + ")"
}

// Data structures

// ArrayLiteral is an array written in the code: [1, 2 + 3, "a"]
type ArrayLiteral struct {
	Token    token.Token // the '[' token
	Elements []Expression
}

func (al *ArrayLiteral) expressionNode()      {}
func (al *ArrayLiteral) TokenLiteral() string { return al.Token.Literal }
func (al *ArrayLiteral) String() string {
	return "[" + joinExpressions(al.Elements) + "]"
}

// IndexExpression reads one element of an array or hash: list[0], ages["ana"]
type IndexExpression struct {
	Token token.Token // the '[' token
	Left  Expression
	Index Expression
}

func (ie *IndexExpression) expressionNode()      {}
func (ie *IndexExpression) TokenLiteral() string { return ie.Token.Literal }
func (ie *IndexExpression) String() string {
	return "(" + ie.Left.String() + "[" + ie.Index.String() + "])"
}

// HashPair is one key: value entry of a HashLiteral.
type HashPair struct {
	Key   Expression
	Value Expression
}

// HashLiteral is a dictionary written in the code: {"name": "Liz", "age": 3}
// Pairs is a slice and not a Go map so the written order is kept.
type HashLiteral struct {
	Token token.Token // the '{' token
	Pairs []HashPair
}

func (hl *HashLiteral) expressionNode()      {}
func (hl *HashLiteral) TokenLiteral() string { return hl.Token.Literal }
func (hl *HashLiteral) String() string {
	pairs := []string{}
	for _, pair := range hl.Pairs {
		pairs = append(pairs, pair.Key.String()+": "+pair.Value.String())
	}
	return "{" + strings.Join(pairs, ", ") + "}"
}

// joinExpressions writes a comma-separated list of expressions.
func joinExpressions(expressions []Expression) string {
	parts := []string{}
	for _, e := range expressions {
		parts = append(parts, e.String())
	}
	return strings.Join(parts, ", ")
}
