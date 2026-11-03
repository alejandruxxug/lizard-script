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
	token.ASSIGN:          ASSIGN,
	token.PLUS_ASSIGN:     ASSIGN,
	token.MINUS_ASSIGN:    ASSIGN,
	token.ASTERISK_ASSIGN: ASSIGN,
	token.SLASH_ASSIGN:    ASSIGN,
	token.OR:              OR,
	token.AND:             AND,
	token.EQ:              EQUALS,
	token.NOT_EQ:          EQUALS,
	token.EXACT:           EQUALS,
	token.NOT_EXACT:       EQUALS,
	token.LT:              LESSGREATER,
	token.GT:              LESSGREATER,
	token.LTE:             LESSGREATER,
	token.GTE:             LESSGREATER,
	token.PLUS:            SUM,
	token.MINUS:           SUM,
	token.ASTERISK:        PRODUCT,
	token.SLASH:           PRODUCT,
	token.DOUBLE_SLASH:    PRODUCT,
	token.MODULE:          PRODUCT,
	token.POW:             EXPONENT,
	token.LPAREN:          CALL,
	token.LBRACKET:        INDEX,
	token.DOT:             INDEX,
}

type (
	prefixParseFn func() ast.Expression
	infixParseFn  func(ast.Expression) ast.Expression
)

// Parser holds the state of one parse.
type Parser struct {
	l       *lexer.Lexer
	errors  []string
	changes []lineChange // see LineChanges

	curToken  token.Token
	peekToken token.Token

	prefixParseFns map[token.TokenType]prefixParseFn
	infixParseFns  map[token.TokenType]infixParseFn

	braceDepth     int         // how many '{' are open, of blocks and hashes
	blockBrace     int         // braceDepth inside the innermost block, 0 at the top level
	parenDepth     int         // how many '(' are open
	brackets       []bracket   // the open '(', '[' and '{', innermost last
	lineIndent     int         // the column of the first token on the line of curToken
	statementStart token.Token // the first token of the statement being parsed
	loopDepth      int         // how many loops are open in the current function
	layerDepth     int         // how many layer bodies are open, for the top level of use
	depth          int         // how many expressions and blocks are being parsed inside each other
	abandoned      bool        // the program nests too deeply and the rest of it was skipped

	// Error recovery, see recovery.go.
	recovering   bool         // an error was found and the statement is being abandoned
	errorLine    int          // the line of the last error, 0 before the first one
	endReported  bool         // an error was found at the end of the input, which the open blocks don't repeat
	closed       bracket      // what the last closer closed, empty when it closed nothing
	keepOpen     bool         // the next closer closes nothing: synchronize found it stray
	blockMissing bool         // the '{' of a block was missing, and no unseen one was opened yet
	blockIndent  int          // the indentation of the line where the innermost block began
	blockErrors  int          // how many errors there were when the innermost block began
	endedEarly   bool         // the block ended before a '}' that closes the one around it
	misaligned   *token.Token // the '{' of the first block closed by a '}' left of its line
	useRejected  *token.Token // the block of the last use that was not at the top level
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
		token.OR:              p.parseInfixExpression,
		token.AND:             p.parseInfixExpression,
		token.EQ:              p.parseInfixExpression,
		token.NOT_EQ:          p.parseInfixExpression,
		token.EXACT:           p.parseInfixExpression,
		token.NOT_EXACT:       p.parseInfixExpression,
		token.LT:              p.parseInfixExpression,
		token.GT:              p.parseInfixExpression,
		token.LTE:             p.parseInfixExpression,
		token.GTE:             p.parseInfixExpression,
		token.PLUS:            p.parseInfixExpression,
		token.MINUS:           p.parseInfixExpression,
		token.ASTERISK:        p.parseInfixExpression,
		token.SLASH:           p.parseInfixExpression,
		token.DOUBLE_SLASH:    p.parseInfixExpression,
		token.MODULE:          p.parseInfixExpression,
		token.POW:             p.parseInfixExpression,
		token.ASSIGN:          p.parseAssignExpression,
		token.PLUS_ASSIGN:     p.parseAssignExpression,
		token.MINUS_ASSIGN:    p.parseAssignExpression,
		token.ASTERISK_ASSIGN: p.parseAssignExpression,
		token.SLASH_ASSIGN:    p.parseAssignExpression,
		token.LPAREN:          p.parseCallExpression,
		token.LBRACKET:        p.parseIndexExpression,
		token.DOT:             p.parseDotExpression,
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
// Once an error is found at the end of the input, the blocks still open
// there are a consequence of it too, so they are not reported.
// An ILLEGAL token is reported as a lexical error, whatever the parser was
// expecting, because that is the stage where the mistake is.
func (p *Parser) errorAt(tok token.Token, format string, args ...any) {
	if p.recovering || p.abandoned || (tok.Type == token.EOF && p.endReported) {
		return
	}
	p.recovering = true
	p.endReported = p.endReported || tok.Type == token.EOF || p.peekTokenIs(token.EOF)

	kind, message := "syntax error", fmt.Sprintf(format, args...)
	if tok.Type == token.ILLEGAL {
		kind, message = "lexical error", illegalMessage(tok)
	}
	p.report(tok, kind+": "+message)
}

// report adds message, at tok, to the errors. An error on the same line as
// the one before is left out: one mistake often breaks the rest of its line,
// and a second, independent one there shows up once the first is fixed.
func (p *Parser) report(tok token.Token, message string) {
	if tok.Line == p.errorLine {
		return
	}
	p.errorLine = tok.Line
	p.errors = append(p.errors, at(tok, message))
}

// at writes message after the position of tok, the way errors show it.
func at(tok token.Token, message string) string {
	return fmt.Sprintf("[line %d, column %d] %s", tok.Line, tok.Column, message)
}

// illegalMessage explains why the lexer rejected tok.
func illegalMessage(tok token.Token) string {
	if strings.HasPrefix(tok.Literal, "\"") {
		return "unterminated string: this \" is never closed"
	}
	if strings.HasPrefix(tok.Literal, "#") && len(tok.Literal) > 1 {
		return "bad color " + tok.Literal
	}
	// A lone backslash outside a string is an unexpected character.
	if strings.HasPrefix(tok.Literal, "\\u") && lexer.ExplainUnicodeEscape(tok.Literal) != "" {
		return lexer.ExplainUnicodeEscape(tok.Literal)
	}
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

// nextToken moves one token forward and keeps the stack of open brackets
// and the indentation of the current line up to date.
func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.l.NextToken()

	if p.curToken.LineStart {
		p.lineIndent = p.curToken.Column
	}
	switch p.curToken.Type {
	case token.LPAREN, token.LBRACKET, token.LBRACE:
		p.open(p.curToken)
	case token.RPAREN, token.RBRACKET, token.RBRACE:
		p.close(p.curToken.Type)
	}
}

// lineEnds reports whether the statement ends at a line break between
// curToken and peekToken. That is the rule of LEP 0001: a statement ends at
// the end of its line, unless the line ends with a token that continues it
// (lexer.ContinuesLine) or a '(', '[' or hash is still open. A block's '{'
// starts statements again, so inside "f(function() { ... })" lines end.
func (p *Parser) lineEnds() bool {
	open := len(p.brackets) > 0 && p.brackets[len(p.brackets)-1].space
	return p.peekToken.LineStart && !open && !lexer.ContinuesLine(p.curToken.Type)
}

// leadingOperatorMessage explains why a line can't start with tok, an
// operator that only works between two values, like + or ==.
func leadingOperatorMessage(tok token.Token) string {
	return fmt.Sprintf("a line can't start with %s; put the %s at the end of the previous line",
		tok.Literal, tok.Literal)
}

// LineChanges lists, as messages with a position, the lines that LEP 0001
// changed: lines that went on with the line before, and now start their own
// statement or are an error. A tool that helps move programs to the new rule
// shows them. It is filled by ParseProgram.
func (p *Parser) LineChanges() []string {
	messages := []string{}
	for _, change := range p.changes {
		messages = append(messages, at(change.tok, change.message))
	}
	return messages
}

// lineChange is a line that LEP 0001 changed: tok starts it.
type lineChange struct {
	tok     token.Token
	message string
}

// noteChange records that tok starts a line that LEP 0001 changed. Each
// level of a nested expression stops at the same line break, so a line
// already noted is skipped.
func (p *Parser) noteChange(tok token.Token, message string) {
	if n := len(p.changes); n > 0 && p.changes[n-1].tok == tok {
		return
	}
	p.changes = append(p.changes, lineChange{tok, message})
}

func (p *Parser) curTokenIs(t token.TokenType) bool {
	return p.curToken.Type == t
}

func (p *Parser) peekTokenIs(t token.TokenType) bool {
	return p.peekToken.Type == t
}

// expectPeek moves to the next token if it has type t. Otherwise it records
// an error, stays where it is and returns false. A number that touches a
// name (the 5 in "puts(5kg)") is the likely cause, so that is what it says.
func (p *Parser) expectPeek(t token.TokenType) bool {
	if p.peekTokenIs(t) {
		p.nextToken()
		return true
	}
	if p.numberTouchesName() {
		p.errorAt(p.curToken, "unknown unit %q", p.peekToken.Literal)
		return false
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
	program := &ast.Program{Statements: p.parseStatementList(token.EOF)}
	if n := len(program.Statements); n > 0 {
		first, last := program.Statements[0].Range(), program.Statements[n-1].Range()
		*program.Range() = ast.Span{StartLine: first.StartLine, StartColumn: first.StartColumn,
			EndLine: last.EndLine, EndColumn: last.EndColumn}
	}
	return program
}

// finish sets the span of node: from the start of first to the end of
// curToken, the last token of node by the rule every parse function keeps.
func (p *Parser) finish(node ast.Node, first token.Token) {
	p.finishAt(node, first.Line, first.Column)
}

// finishAt is finish with the start given as a line and a column.
func (p *Parser) finishAt(node ast.Node, line, column int) {
	*node.Range() = ast.Span{StartLine: line, StartColumn: column,
		EndLine: p.curToken.EndLine, EndColumn: p.curToken.EndColumn}
}

// parseStatementList parses statements until curToken is end or EOF. After a
// broken statement it synchronizes and carries on, so one run reports as many
// independent errors as possible. A broken statement can end on the '}' of
// the block, and then the list ends there too.
func (p *Parser) parseStatementList(end token.TokenType) []ast.Statement {
	statements := []ast.Statement{}

	for !p.curTokenIs(end) && !p.curTokenIs(token.EOF) {
		base := p.statementBase()
		stmt := p.parseStatement()
		if p.recovering {
			p.synchronize(base)
		} else if stmt != nil {
			statements = append(statements, stmt)
		}
		if len(p.brackets) < base {
			break
		}
		if end == token.RBRACE && p.outdentedBrace(base) {
			// The block ends here, and the '}' is left to the block around it.
			p.closeTo(base - 1)
			p.recovering, p.endedEarly = true, true
			break
		}
		p.nextToken()
	}

	return statements
}

// parseStatement parses the statement that starts at curToken and sets its
// span. It returns nil for an empty statement (";") and for a broken one.
func (p *Parser) parseStatement() ast.Statement {
	first, outer := p.curToken, p.statementStart
	p.statementStart = first
	stmt := p.parseStatementKind()
	p.statementStart = outer
	if stmt != nil {
		p.finish(stmt, first)
	}
	return stmt
}

// parseStatementKind picks the kind of statement from curToken.
func (p *Parser) parseStatementKind() ast.Statement {
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
		switch {
		case p.startsLayer():
			return p.parseLayerStatement()
		case p.startsTry():
			return p.parseTryStatement()
		case p.startsContinue():
			return p.parseContinueStatement()
		case p.startsUse():
			return p.parseUseStatement()
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
	if !p.peekEndsStatement() && !p.peekToken.LineStart {
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

	if !p.expectName("let") {
		return nil
	}
	stmt.Name = p.currentIdentifier()

	if p.peekTokenIs(token.COLON) {
		p.errorAt(p.peekToken, "a let gives its value with \"=\", not \":\"")
		return nil
	}
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
// The value starts on the line of the return: a return at the end of its
// line returns nothing.
func (p *Parser) parseReturnStatement() ast.Statement {
	stmt := &ast.ReturnStatement{Token: p.curToken}

	switch {
	case p.peekEndsStatement():
	case p.lineEnds():
		if p.prefixParseFns[p.peekToken.Type] != nil {
			p.noteChange(p.peekToken, "this line is no longer the value of the return before it; "+
				"start the value on the return line")
		}
	default:
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
	p.finish(stmt.Expression, stmt.Token)

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}
	return stmt
}

// startsContinue reports whether the statement that begins at curToken is a
// continue: the name "continue" alone. That is, followed by a ';', a '}', the
// end of the input or the end of its line. Anything else, like "continue + 1"
// or "let continue = 1", is an ordinary expression or a use of an ordinary
// name.
func (p *Parser) startsContinue() bool {
	return p.curToken.Literal == "continue" && (p.peekEndsStatement() || p.lineEnds())
}

// parseContinueStatement parses: continue ;
// Like break, it is an error outside a loop body of the same function.
func (p *Parser) parseContinueStatement() ast.Statement {
	stmt := &ast.ContinueStatement{Token: p.curToken}

	if p.loopDepth == 0 {
		p.errorAt(p.curToken, "continue outside of a loop")
		return nil
	}

	p.endStatement()
	return stmt
}

// startsTry reports whether the statement that begins at curToken is a try:
// the name "try" followed on the same line by '{'. "try {" was never a valid
// statement (the name would be followed by a stray '{'), so nothing that
// worked before reads differently. It is only looked for at the start of a
// statement, so "if try { ... }" still uses a variable named try as the
// condition.
func (p *Parser) startsTry() bool {
	return p.curToken.Literal == "try" && p.peekTokenIs(token.LBRACE) &&
		p.peekToken.Line == p.curToken.Line
}

// parseTryStatement parses: try { ... } catch (e) { ... }
// The name in the parentheses is optional: catch { ... }. Once a try has
// begun the catch is required, so it is read as a name only here, and it
// may start on the line after the '}'. It ends on the '}' of the catch.
func (p *Parser) parseTryStatement() ast.Statement {
	stmt := &ast.TryStatement{Token: p.curToken}

	p.nextToken()
	stmt.Body = p.parseBlockStatement()
	if p.recovering {
		return nil
	}

	if !p.peekTokenIs(token.IDENT) || p.peekToken.Literal != "catch" {
		p.errorAt(p.peekToken, "expected \"catch\" after the try block, got %s", describeToken(p.peekToken))
		return nil
	}
	p.nextToken()

	if p.peekTokenIs(token.LPAREN) {
		p.nextToken()
		if !p.expectPeek(token.IDENT) {
			return nil
		}
		stmt.Name = p.currentIdentifier()
		if !p.expectPeek(token.RPAREN) {
			return nil
		}
	}
	if !p.expectBlockOpen() {
		return nil
	}
	stmt.Catch = p.parseBlockStatement()
	if p.recovering {
		return nil
	}

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}
	return stmt
}

// startsUse reports whether the statement that begins at curToken is a use:
// the name "use" followed on the same line by a string. A name and a string
// side by side were never valid, so nothing that worked before reads
// differently, and "use" anywhere else is an ordinary identifier.
func (p *Parser) startsUse() bool {
	return p.curToken.Literal == "use" && p.peekTokenIs(token.STRING) &&
		p.peekToken.Line == p.curToken.Line
}

// parseUseStatement parses: use "path" ;   use "path" as name ;
// A module's names go into the file's own scope, so a use is only allowed at
// the top level of a file: not in a block, a function or a layer.
func (p *Parser) parseUseStatement() ast.Statement {
	stmt := &ast.UseStatement{Token: p.curToken}

	if p.blockBrace > 0 || p.layerDepth > 0 {
		// A block that is never closed holds every use after it, so each
		// block reports its first one only, and none after an earlier
		// mistake when the use starts its line like a top-level one.
		block := p.innermostBlock()
		if p.useRejected != nil && *p.useRejected == block || len(p.errors) > 0 && p.curToken.Column == 1 {
			p.recovering = true
			return nil
		}
		p.useRejected = &block
		p.errorAt(p.curToken, "use must be at the top level of a file, not inside a block or a layer")
		return nil
	}
	p.nextToken()
	stmt.Path = &ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal}
	p.finish(stmt.Path, p.curToken)
	if stmt.Path.Value == "" {
		p.errorAt(p.curToken, "use needs the path of a module, like use \"shapes\"")
		return nil
	}

	if p.peekTokenIs(token.IDENT) && p.peekToken.Literal == "as" && !p.peekToken.LineStart {
		p.nextToken()
		if !p.peekTokenIs(token.IDENT) || p.peekToken.LineStart {
			p.errorAt(p.peekToken, "expected a name after \"as\", got %s", describeToken(p.peekToken))
			return nil
		}
		p.nextToken()
		stmt.Name = p.currentIdentifier()
	}

	p.endStatement()
	return stmt
}

// innermostBlock returns the '{' of the innermost open block, or an empty
// token outside of any.
func (p *Parser) innermostBlock() token.Token {
	for i := len(p.brackets) - 1; i >= 0; i-- {
		if !p.brackets[i].space {
			return p.brackets[i].tok
		}
	}
	return token.Token{}
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

	p.layerDepth++
	defer func() { p.layerDepth-- }()
	defer p.spanLayerBody(stmt.Body)
	for !p.peekTokenIs(token.EOF) && !p.peekTokenIs(token.RBRACE) &&
		p.peekToken.Line > layer.Line && p.peekToken.Column > layer.Column {
		p.nextToken()
		base := p.statementBase()
		inner := p.parseStatement()
		if p.recovering {
			p.synchronize(base)
		} else if inner != nil {
			stmt.Body.Statements = append(stmt.Body.Statements, inner)
		}
		if len(p.brackets) < base {
			break
		}
	}
	return stmt
}

// spanLayerBody sets the span of a layer's body, which has no braces: from
// its first statement to its last, or empty right after the layer's name.
func (p *Parser) spanLayerBody(body *ast.BlockStatement) {
	statements := body.Statements
	if len(statements) == 0 {
		*body.Range() = ast.Span{StartLine: p.curToken.EndLine, StartColumn: p.curToken.EndColumn,
			EndLine: p.curToken.EndLine, EndColumn: p.curToken.EndColumn}
		return
	}
	first, last := statements[0].Range(), statements[len(statements)-1].Range()
	*body.Range() = ast.Span{StartLine: first.StartLine, StartColumn: first.StartColumn,
		EndLine: last.EndLine, EndColumn: last.EndColumn}
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
		Source:  function.Source,
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

// parseExpressionStatement parses an expression used as a statement. When it
// is broken and starts with a word like "fucntion", the error says which
// keyword was probably meant.
func (p *Parser) parseExpressionStatement() ast.Statement {
	stmt := &ast.ExpressionStatement{Token: p.curToken}
	errors := len(p.errors)

	stmt.Expression = p.parseExpression(LOWEST)
	if stmt.Expression != nil {
		p.endStatement()
	}
	if p.recovering {
		p.misspelledKeyword(stmt.Token, errors)
	}
	if stmt.Expression == nil {
		return nil
	}
	return stmt
}

// parseBlockStatement parses the statements between '{' and '}'. It starts
// on the '{' and ends on the '}'.
//
// A block that is never closed is only found at the end of the input, where
// every '}' after the missing one has closed the wrong block. The first
// block closed by a '}' that starts its line left of the line of its '{' is
// where the '}' was most likely forgotten, so the error goes there. After a
// mistake inside the block, such a '}' is taken as the end of the block
// around it instead (see outdentedBrace).
func (p *Parser) parseBlockStatement() *ast.BlockStatement {
	block := &ast.BlockStatement{Token: p.curToken}
	indent := p.lineIndent

	defer p.leave()
	if !p.enter() {
		return block
	}

	outerBlockBrace, outerIndent, outerErrors := p.blockBrace, p.blockIndent, p.blockErrors
	p.blockBrace, p.blockIndent, p.blockErrors = p.braceDepth, indent, len(p.errors)
	p.nextToken()
	block.Statements = p.parseStatementList(token.RBRACE)
	p.blockBrace, p.blockIndent, p.blockErrors = outerBlockBrace, outerIndent, outerErrors

	switch {
	case p.endedEarly:
		p.endedEarly = false
	case !p.curTokenIs(token.RBRACE) && !p.endReported:
		open := block.Token
		if p.misaligned != nil && p.misaligned.Line > open.Line {
			open = *p.misaligned
		}
		p.errorAt(open, "block is never closed: expected \"}\"")
	case p.misaligned == nil && p.curToken.LineStart && p.curToken.Column < indent:
		p.misaligned = &block.Token
	}

	p.finish(block, block.Token)
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
		p.noExpression()
		return nil
	}
	first := p.curToken
	left := prefix()
	if left != nil {
		p.finish(left, first)
	}

	for left != nil && precedence < p.peekPrecedence() {
		leading := p.lineEnds()
		if leading && !p.joinNextLine() {
			return left
		}
		infix := p.infixParseFns[p.peekToken.Type]
		if infix == nil {
			return left
		}
		start := left.Range()
		p.nextToken()
		if leading && (p.peekToken.LineStart || p.peekTokenIs(token.EOF)) {
			// The operator is alone on its line, so the next line is not its
			// right side: the error is reported, and the operator skipped.
			return left
		}
		left = infix(left)
		if left != nil {
			p.finishAt(left, start.StartLine, start.StartColumn)
		}
	}

	return left
}

// noExpression reports that curToken can't start an expression. A '}' in
// the middle of a line that closed a block begun on an earlier line is more
// likely a mistake than the end of the block, so the block is opened again
// and the '}' closes nothing, as for a ')' or ']' with nothing to close.
func (p *Parser) noExpression() {
	tok := p.curToken
	if b := p.closed; p.closedBlock() && tok.Type == token.RBRACE && !b.unseen &&
		!tok.LineStart && b.tok.Line < tok.Line {
		p.brackets = append(p.brackets, b)
		p.braceDepth++
		p.closed = bracket{}
	}
	switch {
	case p.closed.tok.Type == "" && (tok.Type == token.RPAREN || tok.Type == token.RBRACKET || tok.Type == token.RBRACE):
		p.errorAt(tok, "this %q has nothing to close", tok.Literal)
	default:
		p.errorAt(tok, "expected an expression, got %s", describeToken(tok))
	}
}

// joinNextLine decides what happens to the operator that starts the next
// line, when the expression could go on with it but its line has ended. A
// '-', '(' or '[' starts the next statement, since a statement can begin
// with it, and joinNextLine returns false. Any other operator, like + or ==,
// can't begin anything: that is an error, and joinNextLine returns true so
// the line is read as part of this expression, which is surely what was
// meant. That keeps the parse in step, so the lines after it are still
// checked. Both cases are lines that LEP 0001 changed.
func (p *Parser) joinNextLine() bool {
	operator := p.peekToken
	if p.prefixParseFns[operator.Type] != nil {
		p.noteChange(operator, fmt.Sprintf("this line no longer goes on with the line before: "+
			"%s starts a new statement; to go on, end the line before with an operator or join the lines",
			operator.Literal))
		return false
	}

	message := leadingOperatorMessage(operator)
	p.noteChange(operator, message)
	// Like errorAt, but without abandoning the statement.
	if !p.recovering && !p.abandoned {
		p.report(operator, "syntax error: "+message)
	}
	return true
}

// Prefix functions

func (p *Parser) currentIdentifier() *ast.Identifier {
	identifier := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	p.finish(identifier, p.curToken)
	return identifier
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

	if !p.expectClose(token.RPAREN) {
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

	if !p.expectClose(token.RPAREN) {
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
	if condition == nil || p.assignInCondition(condition) {
		return nil
	}

	if !p.expectBlockOpen() {
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
		return p.parseElseIf(expression)
	}
	if p.peekTokenIs(token.ELSE) {
		p.nextToken()

		// "else if" is the same as "elif".
		if p.peekTokenIs(token.IF) {
			p.nextToken()
			return p.parseElseIf(expression)
		}
		if !p.expectBlockOpen() {
			return nil
		}
		expression.Alternative = p.parseBlockStatement()
	}

	return expression
}

// parseElseIf parses the if that follows an elif or an "else if", starting
// on that keyword, and makes it the whole alternative of expression. It
// returns expression, or nil on error.
func (p *Parser) parseElseIf(expression *ast.IfExpression) ast.Expression {
	keyword := p.curToken

	nested := p.parseIfExpression()
	if nested == nil {
		return nil
	}
	p.finish(nested, keyword)
	statement := &ast.ExpressionStatement{Token: keyword, Expression: nested}
	p.finish(statement, keyword)
	expression.Alternative = &ast.BlockStatement{Token: keyword, Statements: []ast.Statement{statement}}
	p.finish(expression.Alternative, keyword)
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

	if !p.expectBlockOpen() {
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
		first := p.curToken
		if p.curTokenIs(token.LET) {
			expression.Init = p.parseLetStatement()
		} else {
			expression.Init = p.parseExpressionStatement()
		}
		if expression.Init == nil {
			return false
		}
		p.finish(expression.Init, first)
		if !p.curTokenIs(token.SEMICOLON) {
			p.errorAt(p.peekToken, "expected %s, got %s",
				describeType(token.SEMICOLON), describeToken(p.peekToken))
			return false
		}
	}
	p.nextToken()

	if !p.curTokenIs(token.SEMICOLON) {
		expression.Condition = p.parseExpression(LOWEST)
		if expression.Condition == nil || p.assignInCondition(expression.Condition) {
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
		if !p.expectClose(token.RPAREN) {
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

	// The default values run inside the function, so a break or continue
	// there can't reach a loop around it either.
	outerLoopDepth := p.loopDepth
	p.loopDepth = 0
	literal.Parameters, literal.Defaults = p.parseFunctionParameters()
	if literal.Parameters == nil {
		p.loopDepth = outerLoopDepth
		return nil
	}

	if !p.expectBlockOpen() {
		p.loopDepth = outerLoopDepth
		return nil
	}

	literal.Body = p.parseBlockStatement()
	p.loopDepth = outerLoopDepth

	p.finish(literal, function)
	return literal
}

// parseFunctionParameters parses the names between parentheses, each with an
// optional default value: (a, b = 2). It starts on the '(' and ends on the
// ')'. It returns an empty list for "()" and nil on error, so the caller can
// tell them apart. The second list holds the default of each parameter, nil
// for one without it, and is nil when no parameter has a default. A parameter
// without a default can't come after one with it.
func (p *Parser) parseFunctionParameters() ([]*ast.Identifier, []ast.Expression) {
	parameters := []*ast.Identifier{}
	var defaults []ast.Expression

	if p.peekTokenIs(token.RPAREN) {
		p.nextToken()
		return parameters, nil
	}

	for {
		if !p.expectName("function") {
			return nil, nil
		}
		parameter := p.currentIdentifier()

		var value ast.Expression
		if p.peekTokenIs(token.ASSIGN) {
			p.nextToken()
			p.nextToken()
			value = p.parseExpression(LOWEST)
			if value == nil {
				return nil, nil
			}
			if defaults == nil {
				defaults = make([]ast.Expression, len(parameters))
			}
		} else if defaults != nil {
			p.errorAt(p.curToken, "parameter %s needs a default value, it comes after one that has it",
				parameter.Value)
			return nil, nil
		}

		parameters = append(parameters, parameter)
		if defaults != nil {
			defaults = append(defaults, value)
		}

		if !p.peekTokenIs(token.COMMA) {
			break
		}
		p.nextToken()
	}

	if !p.expectClose(token.RPAREN) {
		return nil, nil
	}
	return parameters, defaults
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
//
// A broken hash that starts a statement was not meant as a hash. Right after
// the '{' of a block, as in "if (a) { {", it is most likely one '{' too many,
// so it is forgotten: it doesn't take the '}' of the block. Otherwise it was
// probably meant as a block, so it is skipped as one, with its lines.
func (p *Parser) parseHashLiteral() ast.Expression {
	open := len(p.brackets) - 1
	hash := p.parseHashPairs()
	if hash != nil {
		return hash
	}
	if open < len(p.brackets) && p.brackets[open].tok == p.statementStart {
		if open > 0 && !p.brackets[open-1].space && p.brackets[open-1].tok.Line == p.statementStart.Line {
			p.brackets = append(p.brackets[:open], p.brackets[open+1:]...)
			p.braceDepth--
		} else {
			p.brackets[open].space = false
		}
	}
	return nil
}

// parseHashPairs parses the pairs of a hash and its '}', starting on the '{'.
// It returns nil on error.
func (p *Parser) parseHashPairs() *ast.HashLiteral {
	hash := &ast.HashLiteral{Token: p.curToken, Pairs: []ast.HashPair{}}
	// Inside a hash a line break is just a space, as inside '(' and '['.
	p.brackets[len(p.brackets)-1].space = true

	for !p.peekTokenIs(token.RBRACE) {
		if p.neverClosed() {
			return nil
		}
		p.nextToken()
		// A key stops before '=', so {"a" = 1} gets the error that helps.
		key := p.parseExpression(ASSIGN)
		if key == nil {
			return nil
		}

		if p.peekTokenIs(token.ASSIGN) {
			p.errorAt(p.peekToken, "a hash puts \":\" between a key and its value, not \"=\"")
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

		if !p.peekTokenIs(token.RBRACE) && (p.neverClosed() || !p.expectPeek(token.COMMA)) {
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

// parseAssignExpression parses x = 5 and list[0] = 5, and the compound forms
// x += 5, -=, *= and /=, whose Operator is the character before the '='. Like
// '**', it groups to the right: a = b = 1 is (a = (b = 1)).
func (p *Parser) parseAssignExpression(left ast.Expression) ast.Expression {
	expression := &ast.AssignExpression{Token: p.curToken, Left: left}
	if !p.curTokenIs(token.ASSIGN) {
		expression.Operator = p.curToken.Literal[:1]
	}

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

	if !p.expectClose(token.RBRACKET) {
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

	name := &ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal}
	p.finish(name, p.curToken)
	expression.Index = name
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

	if !p.expectClose(end) {
		return nil
	}
	return list
}
