// Package parser turns the lexer's tokens into an abstract syntax tree.
//
// It is a Pratt parser: each token type can have a prefix function (the token
// starts an expression, like '-' in "-5") and an infix function (the token
// sits between two expressions, like '-' in "a - b"). Each infix token also
// has a precedence, which decides how operators are grouped.
//
// The parser looks at two tokens, curToken and peekToken. Every parse function
// follows the same rule: it starts with curToken on the first token of what it
// parses and ends with curToken on the last one. Keeping that rule is what
// lets the functions call each other.
package parser

import (
	"fmt"
	"lizardscript/ast"
	"lizardscript/lexer"
	"lizardscript/token"
	"math"
	"strconv"
	"strings"
)

// Precedence levels, from weakest to strongest.
const (
	_ int = iota
	LOWEST
	ASSIGN      // x = 5
	OR          // ||
	AND         // &&
	EQUALS      // == != === !==
	LESSGREATER // < > <= >=
	SUM         // + -
	PRODUCT     // * / // %
	PREFIX      // -x !x
	EXPONENT    // ** binds tighter than prefix: -2 ** 2 is -(2 ** 2)
	CALL        // f(x)
	INDEX       // a[i]
)

// maxNestingDepth limits how deeply expressions, blocks and ifs can be
// nested inside each other. The parser and the evaluator call themselves
// once per level, so without a limit a program like ((((...)))) nested a
// million times would overflow Go's stack and crash. Each of those counts as
// a level, so an if inside an if takes about three: the if, its block and
// the statement in it.
const maxNestingDepth = 10000

// precedences lists every token that works as an infix operator. Any other
// token counts as LOWEST, which is what stops parseExpression.
var precedences = map[token.TokenType]int{
	token.ASSIGN:       ASSIGN,
	token.OR:           OR,
	token.AND:          AND,
	token.EQ:           EQUALS,
	token.NOT_EQ:       EQUALS,
	token.EXACT:        EQUALS,
	token.NOT_EXACT:    EQUALS,
	token.LT:           LESSGREATER,
	token.GT:           LESSGREATER,
	token.LTE:          LESSGREATER,
	token.GTE:          LESSGREATER,
	token.PLUS:         SUM,
	token.MINUS:        SUM,
	token.ASTERISK:     PRODUCT,
	token.SLASH:        PRODUCT,
	token.DOUBLE_SLASH: PRODUCT,
	token.MODULE:       PRODUCT,
	token.POW:          EXPONENT,
	token.LPAREN:       CALL,
	token.LBRACKET:     INDEX,
	token.DOT:          INDEX,
}

type (
	prefixParseFn func() ast.Expression
	infixParseFn  func(ast.Expression) ast.Expression
)

// Parser holds the state of one parse.
type Parser struct {
	l      *lexer.Lexer
	errors []string

	curToken  token.Token
	peekToken token.Token

	prefixParseFns map[token.TokenType]prefixParseFn
	infixParseFns  map[token.TokenType]infixParseFn

	recovering bool // an error was found and the statement is being abandoned
	braceDepth int  // how many '{' are open, of blocks and hashes
	blockBrace int  // braceDepth inside the innermost block, 0 at the top level
	parenDepth int  // how many '(' are open
	loopDepth  int  // how many loops are open in the current function
	depth      int  // how many expressions and blocks are being parsed inside each other
	abandoned  bool // the program nests too deeply and the rest of it was skipped
}

// New creates a Parser that reads its tokens from l.
func New(l *lexer.Lexer) *Parser {
	p := &Parser{l: l, errors: []string{}}

	p.prefixParseFns = map[token.TokenType]prefixParseFn{
		token.IDENT:    p.parseIdentifier,
		token.INT:      p.parseIntegerLiteral,
		token.FLOAT:    p.parseFloatLiteral,
		token.COLOR:    p.parseColorLiteral,
		token.STRING:   p.parseStringLiteral,
		token.TRUE:     p.parseBoolean,
		token.FALSE:    p.parseBoolean,
		token.BANG:     p.parsePrefixExpression,
		token.MINUS:    p.parsePrefixExpression,
		token.LPAREN:   p.parseGroupedExpression,
		token.LBRACKET: p.parseArrayLiteral,
		token.LBRACE:   p.parseHashLiteral,
		token.IF:       p.parseIfExpression,
		token.WHILE:    p.parseWhileExpression,
		token.FOR:      p.parseForExpression,
		token.FUNCTION: p.parseFunctionLiteral,
	}

	p.infixParseFns = map[token.TokenType]infixParseFn{
		token.OR:           p.parseInfixExpression,
		token.AND:          p.parseInfixExpression,
		token.EQ:           p.parseInfixExpression,
		token.NOT_EQ:       p.parseInfixExpression,
		token.EXACT:        p.parseInfixExpression,
		token.NOT_EXACT:    p.parseInfixExpression,
		token.LT:           p.parseInfixExpression,
		token.GT:           p.parseInfixExpression,
		token.LTE:          p.parseInfixExpression,
		token.GTE:          p.parseInfixExpression,
		token.PLUS:         p.parseInfixExpression,
		token.MINUS:        p.parseInfixExpression,
		token.ASTERISK:     p.parseInfixExpression,
		token.SLASH:        p.parseInfixExpression,
		token.DOUBLE_SLASH: p.parseInfixExpression,
		token.MODULE:       p.parseInfixExpression,
		token.POW:          p.parseInfixExpression,
		token.ASSIGN:       p.parseAssignExpression,
		token.LPAREN:       p.parseCallExpression,
		token.LBRACKET:     p.parseIndexExpression,
		token.DOT:          p.parseDotExpression,
	}

	// Read two tokens so curToken and peekToken are both set.
	p.nextToken()
	p.nextToken()

	return p
}

// Errors returns the errors found by ParseProgram. If there are any, the tree
// is incomplete and must not be evaluated.
func (p *Parser) Errors() []string {
	return p.errors
}

// errorAt records an error at tok, and from then on ignores the rest of the
// statement: whatever else goes wrong in it is a consequence of this error.
// An ILLEGAL token is reported as a lexical error, whatever the parser was
// expecting, because that is the stage where the mistake is.
func (p *Parser) errorAt(tok token.Token, format string, args ...any) {
	if p.recovering || p.abandoned {
		return
	}
	p.recovering = true

	kind, message := "syntax error", fmt.Sprintf(format, args...)
	if tok.Type == token.ILLEGAL {
		kind, message = "lexical error", illegalMessage(tok)
	}
	p.errors = append(p.errors,
		fmt.Sprintf("[line %d, column %d] %s: %s", tok.Line, tok.Column, kind, message))
}

// illegalMessage explains why the lexer rejected tok.
func illegalMessage(tok token.Token) string {
	if strings.HasPrefix(tok.Literal, "\"") {
		return "unterminated string"
	}
	if strings.HasPrefix(tok.Literal, "#") && len(tok.Literal) > 1 {
		return "bad color " + tok.Literal
	}
	// A lone backslash outside a string is an unexpected character.
	if strings.HasPrefix(tok.Literal, "\\") && len(tok.Literal) > 1 {
		return fmt.Sprintf("unknown escape %s in string", tok.Literal)
	}
	return fmt.Sprintf("unexpected character %q", tok.Literal)
}

// describeToken writes tok the way an error message shows it.
func describeToken(tok token.Token) string {
	switch tok.Type {
	case token.EOF:
		return "end of input"
	case token.IDENT:
		return fmt.Sprintf("identifier %q", tok.Literal)
	default:
		return fmt.Sprintf("%q", tok.Literal)
	}
}

// describeType writes a token type the way an error message shows it.
func describeType(t token.TokenType) string {
	if t == token.IDENT {
		return "identifier"
	}
	return fmt.Sprintf("%q", string(t))
}

// synchronize skips the tokens of a broken statement so parsing can go on
// with the next one. Braces opened by the broken statement are skipped whole,
// so a ';' or '}' inside them doesn't count. Outside of them it stops on a ';'
// or '}', or just before the '}' of the enclosing block, or just before the
// end of input. Stopping there lets the caller's nextToken land on the next
// statement.
func (p *Parser) synchronize() {
	p.recovering = false

	for !p.curTokenIs(token.EOF) {
		if p.braceDepth <= p.blockBrace {
			if p.curTokenIs(token.SEMICOLON) || p.curTokenIs(token.RBRACE) {
				return
			}
			if p.peekTokenIs(token.EOF) || (p.blockBrace > 0 && p.peekTokenIs(token.RBRACE)) {
				return
			}
		}
		p.nextToken()
	}
}

// enter counts one more level of nesting. Its caller must defer leave, even
// when it returns false. Past maxNestingDepth it reports the error and skips
// the rest of the input, so the levels that are open end quietly instead of
// each reporting what is missing.
func (p *Parser) enter() bool {
	p.depth++
	if p.depth <= maxNestingDepth {
		return true
	}

	p.errorAt(p.curToken, "the program is nested too deeply")
	p.abandoned = true
	for !p.curTokenIs(token.EOF) {
		p.nextToken()
	}
	return false
}

func (p *Parser) leave() {
	p.depth--
}

// nextToken moves one token forward and keeps the depth counters up to date.
func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.l.NextToken()

	switch {
	case p.curTokenIs(token.LBRACE):
		p.braceDepth++
	case p.curTokenIs(token.RBRACE) && p.braceDepth > 0:
		p.braceDepth--
	case p.curTokenIs(token.LPAREN):
		p.parenDepth++
	case p.curTokenIs(token.RPAREN) && p.parenDepth > 0:
		p.parenDepth--
	}
}

func (p *Parser) curTokenIs(t token.TokenType) bool {
	return p.curToken.Type == t
}

func (p *Parser) peekTokenIs(t token.TokenType) bool {
	return p.peekToken.Type == t
}

// expectPeek moves to the next token if it has type t. Otherwise it records
// an error, stays where it is and returns false.
func (p *Parser) expectPeek(t token.TokenType) bool {
	if p.peekTokenIs(t) {
		p.nextToken()
		return true
	}
	p.errorAt(p.peekToken, "expected %s, got %s", describeType(t), describeToken(p.peekToken))
	return false
}

func (p *Parser) peekPrecedence() int {
	if precedence, ok := precedences[p.peekToken.Type]; ok {
		return precedence
	}
	return LOWEST
}

func (p *Parser) curPrecedence() int {
	if precedence, ok := precedences[p.curToken.Type]; ok {
		return precedence
	}
	return LOWEST
}

// Statements

// ParseProgram parses the whole input and returns the root of the tree. It
// always returns a Program: check Errors to know whether it is complete.
func (p *Parser) ParseProgram() *ast.Program {
	return &ast.Program{Statements: p.parseStatementList(token.EOF)}
}

// parseStatementList parses statements until curToken is end or EOF. After a
// broken statement it synchronizes and carries on, so one run reports as many
// independent errors as possible.
func (p *Parser) parseStatementList(end token.TokenType) []ast.Statement {
	statements := []ast.Statement{}

	for !p.curTokenIs(end) && !p.curTokenIs(token.EOF) {
		stmt := p.parseStatement()
		if p.recovering {
			p.synchronize()
		} else if stmt != nil {
			statements = append(statements, stmt)
		}
		p.nextToken()
	}

	return statements
}

// parseStatement picks the kind of statement from curToken. It returns nil
// for an empty statement (";") and for a broken one.
func (p *Parser) parseStatement() ast.Statement {
	switch p.curToken.Type {
	case token.SEMICOLON:
		return nil
	case token.LET:
		return p.parseLetStatement()
	case token.RETURN:
		return p.parseReturnStatement()
	case token.BREAK:
		return p.parseBreakStatement()
	case token.IF, token.WHILE, token.FOR:
		return p.parseControlStatement()
	case token.FUNCTION:
		if p.peekTokenIs(token.IDENT) {
			return p.parseFunctionDeclaration()
		}
	case token.IDENT:
		if p.startsLayer() {
			return p.parseLayerStatement()
		}
	}
	return p.parseExpressionStatement()
}

// peekEndsStatement reports whether the next token can only follow the end
// of a statement.
func (p *Parser) peekEndsStatement() bool {
	return p.peekTokenIs(token.SEMICOLON) || p.peekTokenIs(token.RBRACE) || p.peekTokenIs(token.EOF)
}

// endStatement finishes a statement. The ';' is optional before a '}', at the
// end of the input and at the end of a line, but two statements can't share a
// line without it, so "let x = 5 6" is an error and not two statements.
func (p *Parser) endStatement() {
	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
		return
	}
	if !p.peekEndsStatement() && p.peekToken.Line == p.curToken.Line {
		if p.numberTouchesName() {
			p.errorAt(p.curToken, "unknown unit %q", p.peekToken.Literal)
			return
		}
		p.errorAt(p.peekToken, "expected end of statement, got %s", describeToken(p.peekToken))
	}
}

// numberTouchesName reports whether a number is directly followed by a name
// with no space, like 5kg. The lexer reads them as two tokens, since kg is
// not a unit, and this is how the error says what was probably meant.
func (p *Parser) numberTouchesName() bool {
	cur, next := p.curToken, p.peekToken
	return (cur.Type == token.INT || cur.Type == token.FLOAT) && next.Type == token.IDENT &&
		next.Line == cur.Line && next.Column == cur.Column+len(cur.Literal)
}

// parseLetStatement parses: let <name> = <expression> ;
func (p *Parser) parseLetStatement() ast.Statement {
	stmt := &ast.LetStatement{Token: p.curToken}

	if !p.expectPeek(token.IDENT) {
		return nil
	}
	stmt.Name = p.currentIdentifier()

	if !p.expectPeek(token.ASSIGN) {
		return nil
	}
	p.nextToken()

	stmt.Value = p.parseExpression(LOWEST)
	if stmt.Value == nil {
		return nil
	}

	p.endStatement()
	return stmt
}

// parseReturnStatement parses: return ;   return <expression> ;
func (p *Parser) parseReturnStatement() ast.Statement {
	stmt := &ast.ReturnStatement{Token: p.curToken}

	if !p.peekEndsStatement() {
		p.nextToken()
		stmt.ReturnValue = p.parseExpression(LOWEST)
		if stmt.ReturnValue == nil {
			return nil
		}
	}

	p.endStatement()
	return stmt
}

// parseBreakStatement parses: break ;
// Using it outside a loop is an error here and not at run time, because the
// structure of the program alone shows the mistake.
func (p *Parser) parseBreakStatement() ast.Statement {
	stmt := &ast.BreakStatement{Token: p.curToken}

	if p.loopDepth == 0 {
		p.errorAt(p.curToken, "break outside of a loop")
		return nil
	}

	p.endStatement()
	return stmt
}

// parseControlStatement parses an if, while or for that starts a statement.
// It calls the prefix function directly, skipping the infix loop of
// parseExpression: a statement ends at its closing '}', so in
//
//	if (a) { 1 }
//	-5;
//
// the '-' starts a new statement and is not a subtraction from the if.
func (p *Parser) parseControlStatement() ast.Statement {
	stmt := &ast.ExpressionStatement{Token: p.curToken}

	stmt.Expression = p.prefixParseFns[p.curToken.Type]()
	if stmt.Expression == nil {
		return nil
	}

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}
	return stmt
}

// startsLayer reports whether the statement that begins at curToken is a
// layer: the name "layer" followed on the same line by a name or a string.
// Two names side by side were never valid, so nothing that worked before
// reads differently, and "layer" anywhere else is an ordinary identifier.
func (p *Parser) startsLayer() bool {
	if p.curToken.Literal != "layer" || p.peekToken.Line != p.curToken.Line {
		return false
	}
	return p.peekTokenIs(token.IDENT) || p.peekTokenIs(token.STRING)
}

// parseLayerStatement parses: layer <name> ;?  followed by an indented body.
// The body is every following statement that starts on a later line, in a
// column greater than the one of "layer". Columns are the ones the lexer
// reports, where a tab counts as one. The body ends at the first statement
// that is not indented more, at a '}' or at the end of the input.
func (p *Parser) parseLayerStatement() ast.Statement {
	layer := p.curToken
	p.nextToken()
	stmt := &ast.LayerStatement{Token: layer, Name: p.curToken.Literal}
	stmt.Body = &ast.BlockStatement{Token: layer}

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	} else if !p.peekEndsStatement() && p.peekToken.Line == layer.Line {
		p.errorAt(p.peekToken, "a layer is \"layer name\" alone on its line, got %s after the name",
			describeToken(p.peekToken))
		return nil
	}

	defer p.leave()
	if !p.enter() {
		return stmt
	}

	for !p.peekTokenIs(token.EOF) && !p.peekTokenIs(token.RBRACE) &&
		p.peekToken.Line > layer.Line && p.peekToken.Column > layer.Column {
		p.nextToken()
		inner := p.parseStatement()
		if p.recovering {
			p.synchronize()
			if p.braceDepth < p.blockBrace {
				break
			}
		} else if inner != nil {
			stmt.Body.Statements = append(stmt.Body.Statements, inner)
		}
	}
	return stmt
}

// parseFunctionDeclaration parses: function <name>(<parameters>) { ... }
// It builds the same tree as let <name> = function(<parameters>) { ... };
// so the evaluator doesn't need anything extra for it.
func (p *Parser) parseFunctionDeclaration() ast.Statement {
	function := p.curToken
	stmt := &ast.LetStatement{Token: token.Token{
		Type:    token.LET,
		Literal: "let",
		Line:    function.Line,
		Column:  function.Column,
	}}

	p.nextToken()
	stmt.Name = p.currentIdentifier()

	stmt.Value = p.parseFunction(function)
	if stmt.Value == nil {
		return nil
	}

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}
	return stmt
}

func (p *Parser) parseExpressionStatement() ast.Statement {
	stmt := &ast.ExpressionStatement{Token: p.curToken}

	stmt.Expression = p.parseExpression(LOWEST)
	if stmt.Expression == nil {
		return nil
	}

	p.endStatement()
	return stmt
}

// parseBlockStatement parses the statements between '{' and '}'. It starts
// on the '{' and ends on the '}'.
func (p *Parser) parseBlockStatement() *ast.BlockStatement {
	block := &ast.BlockStatement{Token: p.curToken}

	defer p.leave()
	if !p.enter() {
		return block
	}

	outerBlockBrace := p.blockBrace
	p.blockBrace = p.braceDepth
	p.nextToken()
	block.Statements = p.parseStatementList(token.RBRACE)
	p.blockBrace = outerBlockBrace

	if !p.curTokenIs(token.RBRACE) {
		p.errorAt(block.Token, "block is never closed: expected \"}\"")
	}

	return block
}

// Expressions

// parseExpression is the core of the Pratt parser. It parses one expression
// made of operators that bind tighter than precedence, the precedence of the
// operator on its left.
//
// With "1 + 2 * 3" and LOWEST: the prefix function reads 1, and since '+' is
// stronger than LOWEST, its infix function takes 1 as its left side and calls
// parseExpression(SUM) for the right side. That call reads 2 and absorbs '*'
// because PRODUCT > SUM, giving (1 + (2 * 3)). With "1 * 2 + 3" the call is
// parseExpression(PRODUCT), it stops at '+', and the result is ((1 * 2) + 3).
func (p *Parser) parseExpression(precedence int) ast.Expression {
	defer p.leave()
	if !p.enter() {
		return nil
	}

	prefix := p.prefixParseFns[p.curToken.Type]
	if prefix == nil {
		p.errorAt(p.curToken, "expected an expression, got %s", describeToken(p.curToken))
		return nil
	}
	left := prefix()

	for left != nil && precedence < p.peekPrecedence() {
		infix := p.infixParseFns[p.peekToken.Type]
		if infix == nil {
			return left
		}
		p.nextToken()
		left = infix(left)
	}

	return left
}

// Prefix functions

func (p *Parser) currentIdentifier() *ast.Identifier {
	return &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
}

func (p *Parser) parseIdentifier() ast.Expression {
	return p.currentIdentifier()
}

// parseIntegerLiteral converts the token's digits to a number. The base is
// always 10: with base 0, strconv would read 010 as octal and reject 08.
func (p *Parser) parseIntegerLiteral() ast.Expression {
	value, err := strconv.ParseInt(p.curToken.Literal, 10, 64)
	if err != nil {
		p.errorAt(p.curToken, "integer %s is too large", p.curToken.Literal)
		return nil
	}
	return &ast.IntegerLiteral{Token: p.curToken, Value: value}
}

// parseFloatLiteral converts the token's text to a number. A number too big
// for a float is an error, since lizard-script has no infinity.
func (p *Parser) parseFloatLiteral() ast.Expression {
	value, err := strconv.ParseFloat(p.curToken.Literal, 64)
	if err != nil || math.IsInf(value, 0) {
		p.errorAt(p.curToken, "number %s is too large", p.curToken.Literal)
		return nil
	}
	return &ast.FloatLiteral{Token: p.curToken, Value: value}
}

// parseColorLiteral reads the hex digits of a color token. The lexer already
// checked that there are 3, 6 or 8 of them. #rgb stands for #rrggbb.
func (p *Parser) parseColorLiteral() ast.Expression {
	digits := p.curToken.Literal[1:]
	if len(digits) == 3 {
		digits = digits[0:1] + digits[0:1] + digits[1:2] + digits[1:2] + digits[2:3] + digits[2:3]
	}
	channels := []uint8{0, 0, 0, 255}
	for i := 0; i < len(digits); i += 2 {
		value, _ := strconv.ParseUint(digits[i:i+2], 16, 8)
		channels[i/2] = uint8(value)
	}
	return &ast.ColorLiteral{Token: p.curToken,
		R: channels[0], G: channels[1], B: channels[2], A: channels[3]}
}

func (p *Parser) parseStringLiteral() ast.Expression {
	return &ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal}
}

func (p *Parser) parseBoolean() ast.Expression {
	return &ast.Boolean{Token: p.curToken, Value: p.curTokenIs(token.TRUE)}
}

// parsePrefixExpression parses -x and !x. The operand is parsed with PREFIX,
// so in "-a * b" the '-' only takes a: ((-a) * b).
func (p *Parser) parsePrefixExpression() ast.Expression {
	expression := &ast.PrefixExpression{Token: p.curToken, Operator: p.curToken.Literal}

	p.nextToken()

	expression.Right = p.parseExpression(PREFIX)
	if expression.Right == nil {
		return nil
	}
	return expression
}

// parseGroupedExpression parses (x), or a point (x, y) and (x, y, z).
// Parentheses around one expression create no node: their only effect is to
// restart the precedence from LOWEST. A comma after the first expression
// makes it a point.
func (p *Parser) parseGroupedExpression() ast.Expression {
	open := p.curToken
	p.nextToken()

	// () is not an empty group: say what was probably meant.
	if p.curTokenIs(token.RPAREN) {
		p.errorAt(open, pointSizeMessage)
		return nil
	}

	expression := p.parseExpression(LOWEST)
	if expression == nil {
		return nil
	}

	if p.peekTokenIs(token.COMMA) {
		return p.parsePointRest(open, expression)
	}

	if !p.expectPeek(token.RPAREN) {
		return nil
	}
	return expression
}

// pointSizeMessage is the error of a point with no coordinates or too many.
const pointSizeMessage = "a point has two or three coordinates"

// parsePointRest parses the rest of a point after its first coordinate,
// starting on that coordinate and ending on the ')'.
func (p *Parser) parsePointRest(open token.Token, first ast.Expression) ast.Expression {
	point := &ast.PointLiteral{Token: open, Coords: []ast.Expression{first}}

	for p.peekTokenIs(token.COMMA) {
		p.nextToken()
		p.nextToken()
		coord := p.parseExpression(LOWEST)
		if coord == nil {
			return nil
		}
		point.Coords = append(point.Coords, coord)
	}

	if !p.expectPeek(token.RPAREN) {
		return nil
	}
	if len(point.Coords) > 3 {
		p.errorAt(open, pointSizeMessage)
		return nil
	}
	return point
}

// parseCondition parses the condition of an if, elif or while. It starts on
// the keyword and ends on the '{' of the body. The parentheses are just a
// grouped expression, so they are optional.
func (p *Parser) parseCondition() ast.Expression {
	p.nextToken()

	condition := p.parseExpression(LOWEST)
	if condition == nil {
		return nil
	}

	if !p.expectPeek(token.LBRACE) {
		return nil
	}
	return condition
}

// parseIfExpression parses an if with any number of elif and an optional
// else. It also parses each elif, as an if that lives inside the else block
// of the previous one. It ends on the last '}'.
func (p *Parser) parseIfExpression() ast.Expression {
	expression := &ast.IfExpression{Token: p.curToken}

	// Each elif is an if inside the previous one, so it is a level too.
	defer p.leave()
	if !p.enter() {
		return nil
	}

	expression.Condition = p.parseCondition()
	if expression.Condition == nil {
		return nil
	}

	expression.Consequence = p.parseBlockStatement()

	if p.peekTokenIs(token.ELIF) {
		p.nextToken()
		elif := p.curToken

		nested := p.parseIfExpression()
		if nested == nil {
			return nil
		}

		expression.Alternative = &ast.BlockStatement{
			Token: elif,
			Statements: []ast.Statement{
				&ast.ExpressionStatement{Token: elif, Expression: nested},
			},
		}
	} else if p.peekTokenIs(token.ELSE) {
		p.nextToken()

		if !p.expectPeek(token.LBRACE) {
			return nil
		}
		expression.Alternative = p.parseBlockStatement()
	}

	return expression
}

// parseLoopBody parses the body of a loop, the only place where break is
// allowed. It starts on the '{' and ends on the '}'.
func (p *Parser) parseLoopBody() *ast.BlockStatement {
	p.loopDepth++
	body := p.parseBlockStatement()
	p.loopDepth--
	return body
}

// parseWhileExpression parses: while (<condition>) { ... }
func (p *Parser) parseWhileExpression() ast.Expression {
	expression := &ast.WhileExpression{Token: p.curToken}

	expression.Condition = p.parseCondition()
	if expression.Condition == nil {
		return nil
	}

	expression.Body = p.parseLoopBody()
	return expression
}

// parseForExpression parses: for (<init>; <condition>; <update>) { ... }
// A broken header is skipped first, because its ';' would otherwise look like
// the end of the statement to synchronize.
func (p *Parser) parseForExpression() ast.Expression {
	expression := &ast.ForExpression{Token: p.curToken}
	outerParenDepth := p.parenDepth

	if !p.parseForHeader(expression) {
		for p.parenDepth > outerParenDepth && !p.peekTokenIs(token.LBRACE) && !p.curTokenIs(token.EOF) {
			p.nextToken()
		}
		return nil
	}

	if !p.expectPeek(token.LBRACE) {
		return nil
	}
	expression.Body = p.parseLoopBody()
	return expression
}

// parseForHeader parses (<init>; <condition>; <update>), ending on the ')'.
// Any of the three parts can be left out. <init> is a let or an expression
// statement, which consumes its own ';'. The other two are expressions, which
// don't. It returns false on error.
func (p *Parser) parseForHeader(expression *ast.ForExpression) bool {
	if !p.expectPeek(token.LPAREN) {
		return false
	}
	p.nextToken()

	if !p.curTokenIs(token.SEMICOLON) {
		if p.curTokenIs(token.LET) {
			expression.Init = p.parseLetStatement()
		} else {
			expression.Init = p.parseExpressionStatement()
		}
		if expression.Init == nil {
			return false
		}
		if !p.curTokenIs(token.SEMICOLON) {
			p.errorAt(p.peekToken, "expected %s, got %s",
				describeType(token.SEMICOLON), describeToken(p.peekToken))
			return false
		}
	}
	p.nextToken()

	if !p.curTokenIs(token.SEMICOLON) {
		expression.Condition = p.parseExpression(LOWEST)
		if expression.Condition == nil {
			return false
		}
		if !p.expectPeek(token.SEMICOLON) {
			return false
		}
	}
	p.nextToken()

	if !p.curTokenIs(token.RPAREN) {
		expression.Update = p.parseExpression(LOWEST)
		if expression.Update == nil {
			return false
		}
		if !p.expectPeek(token.RPAREN) {
			return false
		}
	}
	return true
}

// parseFunctionLiteral parses an anonymous function: function(a, b) { ... }
func (p *Parser) parseFunctionLiteral() ast.Expression {
	return p.parseFunction(p.curToken)
}

// parseFunction parses the parameters and body of a function, from the token
// before the '('. A function body starts outside of any loop, so a break
// can't jump out of a function into the loop that defined it.
func (p *Parser) parseFunction(function token.Token) ast.Expression {
	literal := &ast.FunctionLiteral{Token: function}

	if !p.expectPeek(token.LPAREN) {
		return nil
	}

	literal.Parameters = p.parseFunctionParameters()
	if literal.Parameters == nil {
		return nil
	}

	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	outerLoopDepth := p.loopDepth
	p.loopDepth = 0
	literal.Body = p.parseBlockStatement()
	p.loopDepth = outerLoopDepth

	return literal
}

// parseFunctionParameters parses the names between parentheses. It starts on
// the '(' and ends on the ')'. It returns an empty list for "()" and nil on
// error, so the caller can tell them apart.
func (p *Parser) parseFunctionParameters() []*ast.Identifier {
	parameters := []*ast.Identifier{}

	if p.peekTokenIs(token.RPAREN) {
		p.nextToken()
		return parameters
	}

	if !p.expectPeek(token.IDENT) {
		return nil
	}
	parameters = append(parameters, p.currentIdentifier())

	for p.peekTokenIs(token.COMMA) {
		p.nextToken()
		if !p.expectPeek(token.IDENT) {
			return nil
		}
		parameters = append(parameters, p.currentIdentifier())
	}

	if !p.expectPeek(token.RPAREN) {
		return nil
	}
	return parameters
}

func (p *Parser) parseArrayLiteral() ast.Expression {
	array := &ast.ArrayLiteral{Token: p.curToken}

	array.Elements = p.parseExpressionList(token.RBRACKET)
	if array.Elements == nil {
		return nil
	}
	return array
}

// parseHashLiteral parses {key: value, ...}. It starts on the '{' and ends on
// the '}'. Keys and values can be any expression, and a trailing comma is
// allowed. A '{' can only start a hash, because blocks exist only as the body
// of an if, loop or function.
func (p *Parser) parseHashLiteral() ast.Expression {
	hash := &ast.HashLiteral{Token: p.curToken, Pairs: []ast.HashPair{}}

	for !p.peekTokenIs(token.RBRACE) {
		p.nextToken()
		key := p.parseExpression(LOWEST)
		if key == nil {
			return nil
		}

		if !p.expectPeek(token.COLON) {
			return nil
		}

		p.nextToken()
		value := p.parseExpression(LOWEST)
		if value == nil {
			return nil
		}

		hash.Pairs = append(hash.Pairs, ast.HashPair{Key: key, Value: value})

		if !p.peekTokenIs(token.RBRACE) && !p.expectPeek(token.COMMA) {
			return nil
		}
	}

	p.nextToken()
	return hash
}

// Infix functions

// parseInfixExpression parses a binary operator. The right side uses the
// operator's own precedence, so an equal operator is not absorbed and
// "1 + 2 + 3" groups to the left: ((1 + 2) + 3). '**' groups to the right, so
// it uses one level less: 2 ** 3 ** 2 is (2 ** (3 ** 2)).
func (p *Parser) parseInfixExpression(left ast.Expression) ast.Expression {
	expression := &ast.InfixExpression{
		Token:    p.curToken,
		Left:     left,
		Operator: p.curToken.Literal,
	}

	precedence := p.curPrecedence()
	if p.curTokenIs(token.POW) {
		precedence--
	}

	p.nextToken()

	expression.Right = p.parseExpression(precedence)
	if expression.Right == nil {
		return nil
	}
	return expression
}

// parseAssignExpression parses x = 5 and list[0] = 5. Like '**', it groups to
// the right: a = b = 1 is (a = (b = 1)).
func (p *Parser) parseAssignExpression(left ast.Expression) ast.Expression {
	expression := &ast.AssignExpression{Token: p.curToken, Left: left}

	switch left.(type) {
	case *ast.Identifier, *ast.IndexExpression:
	default:
		p.errorAt(p.curToken, "cannot assign to %s", left.String())
		return nil
	}

	p.nextToken()

	expression.Value = p.parseExpression(ASSIGN - 1)
	if expression.Value == nil {
		return nil
	}
	return expression
}

// parseCallExpression parses f(a, b), with '(' working as an infix operator
// whose left side is the function.
func (p *Parser) parseCallExpression(function ast.Expression) ast.Expression {
	expression := &ast.CallExpression{Token: p.curToken, Function: function}

	expression.Arguments = p.parseExpressionList(token.RPAREN)
	if expression.Arguments == nil {
		return nil
	}
	return expression
}

// parseIndexExpression parses a[i], with '[' working as an infix operator.
func (p *Parser) parseIndexExpression(left ast.Expression) ast.Expression {
	expression := &ast.IndexExpression{Token: p.curToken, Left: left}

	p.nextToken()

	expression.Index = p.parseExpression(LOWEST)
	if expression.Index == nil {
		return nil
	}

	if !p.expectPeek(token.RBRACKET) {
		return nil
	}
	return expression
}

// parseDotExpression parses a.b as a["b"]: the same tree, so the evaluator
// doesn't know the difference. The name after the dot must be an identifier.
func (p *Parser) parseDotExpression(left ast.Expression) ast.Expression {
	expression := &ast.IndexExpression{Token: p.curToken, Left: left}

	if !p.peekTokenIs(token.IDENT) {
		p.errorAt(p.peekToken, "expected a name after \".\"")
		return nil
	}
	p.nextToken()

	expression.Index = &ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal}
	return expression
}

// parseExpressionList parses comma-separated expressions up to end, for call
// arguments and array elements. It starts on the opening token and ends on
// end. It returns an empty list when there are no elements and nil on error.
func (p *Parser) parseExpressionList(end token.TokenType) []ast.Expression {
	list := []ast.Expression{}

	if p.peekTokenIs(end) {
		p.nextToken()
		return list
	}

	p.nextToken()
	first := p.parseExpression(LOWEST)
	if first == nil {
		return nil
	}
	list = append(list, first)

	for p.peekTokenIs(token.COMMA) {
		p.nextToken()
		p.nextToken()
		next := p.parseExpression(LOWEST)
		if next == nil {
			return nil
		}
		list = append(list, next)
	}

	if !p.expectPeek(end) {
		return nil
	}
	return list
}
