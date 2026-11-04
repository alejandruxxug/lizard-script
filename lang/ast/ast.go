// Package ast defines the nodes of the abstract syntax tree that the parser
// builds. The tree keeps the structure of a program and drops details like
// parentheses and semicolons, so "(1 + 2)" and "1 + 2" produce the same tree.
package ast

import (
	"bytes"
	"github.com/alejandruxxug/lizard-script/lang/token"
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
	// Range gives where the node is in the source, to read or set it.
	Range() *Span
}

// Span is where a node is in the source: from the first character of its
// first token (StartLine, StartColumn) to just after the last character of
// its last token (EndLine, EndColumn), counted like token positions. A node
// written in parentheses includes them, so in "(1 + 2) * 3" the sum spans
// "(1 + 2)". The parser fills it in; a node built by hand has zeros.
//
// Every node embeds a Span. Its Token is not always its first token: the
// Token of "a + b" is the "+".
type Span struct {
	StartLine, StartColumn int
	EndLine, EndColumn     int
}

// Range returns the span itself, so the parser can fill it in.
func (s *Span) Range() *Span { return s }

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
	Span
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
	Span
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
	Span
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
	Span
	Token token.Token // the BREAK token
}

func (bs *BreakStatement) statementNode()       {}
func (bs *BreakStatement) TokenLiteral() string { return bs.Token.Literal }
func (bs *BreakStatement) String() string       { return bs.TokenLiteral() + ";" }

// ContinueStatement skips the rest of the current round of the innermost
// loop: continue;
// The parser only accepts it inside a while or for body. In the source it is
// the plain name "continue", not a keyword, so a variable of that name works
// everywhere else.
type ContinueStatement struct {
	Span
	Token token.Token // the "continue" name
}

func (cs *ContinueStatement) statementNode()       {}
func (cs *ContinueStatement) TokenLiteral() string { return cs.Token.Literal }
func (cs *ContinueStatement) String() string       { return cs.TokenLiteral() + ";" }

// TryStatement runs Body and, if a runtime error stops it, runs Catch with
// the error bound to Name: try { ... } catch (e) { ... }
// Name is nil for "catch { ... }". In the source "try" and "catch" are plain
// names, only read as a try statement in the place described in the parser.
type TryStatement struct {
	Span
	Token token.Token // the "try" name
	Body  *BlockStatement
	Name  *Identifier // nil when the catch has no name
	Catch *BlockStatement
}

func (ts *TryStatement) statementNode()       {}
func (ts *TryStatement) TokenLiteral() string { return ts.Token.Literal }
func (ts *TryStatement) String() string {
	catch := "catch "
	if ts.Name != nil {
		catch += "(" + ts.Name.String() + ") "
	}
	return "try " + ts.Body.String() + " " + catch + ts.Catch.String()
}

// ExpressionStatement lets an expression stand alone as a statement, because
// a program is a list of statements: add(1, 2);
type ExpressionStatement struct {
	Span
	Token      token.Token // the first token of the expression
	Expression Expression
}

func (es *ExpressionStatement) statementNode()       {}
func (es *ExpressionStatement) TokenLiteral() string { return es.Token.Literal }
func (es *ExpressionStatement) String() string       { return es.Expression.String() }

// BlockStatement is a list of statements between braces: the body of an if,
// a loop or a function.
type BlockStatement struct {
	Span
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

// LayerStatement is a named group of statements, written as "layer name"
// followed by an indented body. The body is a BlockStatement whose Token is
// the "layer" token, since there is no brace.
type LayerStatement struct {
	Span
	Token token.Token // the "layer" identifier
	Name  string
	Body  *BlockStatement
}

func (ls *LayerStatement) statementNode()       {}
func (ls *LayerStatement) TokenLiteral() string { return ls.Token.Literal }

// String prints "layer name" and then each statement of the body on its own
// line, indented by two spaces more than the layer.
func (ls *LayerStatement) String() string {
	var out bytes.Buffer
	out.WriteString("layer ")
	if isPlainName(ls.Name) {
		out.WriteString(ls.Name)
	} else {
		out.WriteString("\"" + ls.Name + "\"")
	}
	for _, s := range ls.Body.Statements {
		out.WriteString("\n")
		out.WriteString("  " + strings.ReplaceAll(s.String(), "\n", "\n  "))
	}
	return out.String()
}

// UseStatement loads a module, the file at Path, and makes its names
// available: use "draw/shapes" lets the file use every name the module
// defines, and use "draw/shapes" as shapes puts them in a hash called
// shapes instead. In the source "use" and "as" are plain names, read as a
// use statement only at the top level of a file (see the parser).
type UseStatement struct {
	Span
	Token token.Token    // the "use" name
	Path  *StringLiteral // the module, as written: "draw/shapes"
	Name  *Identifier    // the name after "as", or nil
}

func (us *UseStatement) statementNode()       {}
func (us *UseStatement) TokenLiteral() string { return us.Token.Literal }
func (us *UseStatement) String() string {
	if us.Name == nil {
		return "use " + us.Path.String() + ";"
	}
	return "use " + us.Path.String() + " as " + us.Name.String() + ";"
}

// isPlainName reports whether name can be written without quotes.
func isPlainName(name string) bool {
	if name == "" {
		return false
	}
	for i, c := range name {
		letter := c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
		if !letter && (i == 0 || c < '0' || c > '9') {
			return false
		}
	}
	return token.LookupIdent(name) == token.IDENT
}

// Leaf expressions

// Identifier is the name of a variable or function.
type Identifier struct {
	Span
	Token token.Token // the IDENT token
	Value string
}

func (i *Identifier) expressionNode()      {}
func (i *Identifier) TokenLiteral() string { return i.Token.Literal }
func (i *Identifier) String() string       { return i.Value }

// IntegerLiteral is a whole number written in the code.
type IntegerLiteral struct {
	Span
	Token token.Token // the INT token
	Value int64
}

func (il *IntegerLiteral) expressionNode()      {}
func (il *IntegerLiteral) TokenLiteral() string { return il.Token.Literal }
func (il *IntegerLiteral) String() string       { return il.Token.Literal }

// FloatLiteral is a decimal number written in the code. A number with a unit
// like 45deg is one too, already converted by the lexer.
type FloatLiteral struct {
	Span
	Token token.Token // the FLOAT token
	Value float64
}

func (fl *FloatLiteral) expressionNode()      {}
func (fl *FloatLiteral) TokenLiteral() string { return fl.Token.Literal }
func (fl *FloatLiteral) String() string       { return fl.Token.Literal }

// ColorLiteral is a color written in the code, like #3fa34d. Each channel is
// 0 to 255; A is 255 when the literal has no alpha.
type ColorLiteral struct {
	Span
	Token      token.Token // the COLOR token
	R, G, B, A uint8
}

func (cl *ColorLiteral) expressionNode()      {}
func (cl *ColorLiteral) TokenLiteral() string { return cl.Token.Literal }
func (cl *ColorLiteral) String() string       { return cl.Token.Literal }

// PointLiteral is (x, y) or (x, y, z): two or three coordinates.
type PointLiteral struct {
	Span
	Token  token.Token // the '(' token
	Coords []Expression
}

func (pl *PointLiteral) expressionNode()      {}
func (pl *PointLiteral) TokenLiteral() string { return pl.Token.Literal }
func (pl *PointLiteral) String() string {
	parts := []string{}
	for _, c := range pl.Coords {
		parts = append(parts, c.String())
	}
	return "(" + strings.Join(parts, ", ") + ")"
}

// Boolean is true or false.
type Boolean struct {
	Span
	Token token.Token // the TRUE or FALSE token
	Value bool
}

func (b *Boolean) expressionNode()      {}
func (b *Boolean) TokenLiteral() string { return b.Token.Literal }
func (b *Boolean) String() string       { return b.Token.Literal }

// StringLiteral is a text between double quotes. Value has no quotes.
type StringLiteral struct {
	Span
	Token token.Token // the STRING token
	Value string
}

func (sl *StringLiteral) expressionNode()      {}
func (sl *StringLiteral) TokenLiteral() string { return sl.Token.Literal }
func (sl *StringLiteral) String() string       { return "\"" + sl.Value + "\"" }

// Operators

// PrefixExpression is an operator before its operand: -x, !done
type PrefixExpression struct {
	Span
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
	Span
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
// For x += 1 the Operator is "+" (also "-", "*" and "/"), and it means
// x = x + 1 with the target evaluated once. It is "" for a plain '='.
type AssignExpression struct {
	Span
	Token    token.Token // the '=' token, or '+=' and its kin
	Left     Expression
	Operator string
	Value    Expression
}

func (ae *AssignExpression) expressionNode()      {}
func (ae *AssignExpression) TokenLiteral() string { return ae.Token.Literal }
func (ae *AssignExpression) String() string {
	return "(" + ae.Left.String() + " " + ae.Operator + "= " + ae.Value.String() + ")"
}

// Control flow

// IfExpression is a conditional. Alternative is nil when there is no else.
// There is no elif node: "if (a) {} elif (b) {}" is built as
// "if (a) {} else { if (b) {} }", with elif as the Token of the inner if.
type IfExpression struct {
	Span
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
	Span
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
	Span
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
//
// Defaults has one entry per parameter: the expression of its default value,
// or nil for a parameter without one. It is nil when no parameter has a
// default.
type FunctionLiteral struct {
	Span
	Token      token.Token // the FUNCTION token
	Parameters []*Identifier
	Defaults   []Expression
	Body       *BlockStatement
}

func (fl *FunctionLiteral) expressionNode()      {}
func (fl *FunctionLiteral) TokenLiteral() string { return fl.Token.Literal }
func (fl *FunctionLiteral) String() string {
	params := []string{}
	for i, p := range fl.Parameters {
		params = append(params, ParameterString(p, fl.Defaults, i))
	}
	return fl.TokenLiteral() + "(" + strings.Join(params, ", ") + ") " + fl.Body.String()
}

// ParameterString writes parameter i the way the source does: "b" or "b = 2".
func ParameterString(parameter *Identifier, defaults []Expression, i int) string {
	if i < len(defaults) && defaults[i] != nil {
		return parameter.String() + " = " + defaults[i].String()
	}
	return parameter.String()
}

// CallExpression calls a function: add(1, 2 * 3)
// Function is any expression that produces a function.
type CallExpression struct {
	Span
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
	Span
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
	Span
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
	Span
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
