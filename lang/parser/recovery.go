package parser

import (
	"github.com/alejandruxxug/lizard-script/lang/lexer"
	"github.com/alejandruxxug/lizard-script/lang/token"
)

// Error recovery. After a syntax error the parser skips the rest of the
// broken statement and goes on with the next one, so one run reports every
// independent mistake, each once, and the good lines around them none.
//
// Where a statement ends is the rule of LEP 0001: at the end of its line,
// unless the line goes on or a bracket is still open. The parser keeps a
// stack of the open brackets, each with the indentation of its line, so
// after an error it knows which brackets the broken statement opened, skips
// them whole, and stops at the end of the line, at a ';', or before the '}'
// of the block around it. A bracket that is never closed would hide the rest
// of the program, so the stack also tells when a line can't be inside it and
// the bracket must be missing its closer: that line starts a new statement,
// and the error says where the bracket was opened.
//
// A '{' that the program forgot is opened unseen, so the body and the '}'
// after it are skipped with the broken statement, and a '}' that a mistake
// left in the middle of a line doesn't end the block around it.

// bracket is an open '(', '[' or '{' on the stack of the parser.
type bracket struct {
	tok    token.Token // the opening token
	indent int         // the column of the first token on its line
	space  bool        // a line break inside it is just a space: true for '(', '[' and a hash
	unseen bool        // a block '{' that the program forgot, opened by expectBlockOpen
}

// synchronize skips the tokens of a broken statement so parsing can go on
// with the next one. base is how many brackets were open when the statement
// began. The brackets it opened are skipped whole, so a ';' or a line break
// inside them doesn't count. Outside of them it stops on a ';' or on the '}'
// of a block the statement opened (unless an else, elif or catch follows),
// at the end of the line (LEP 0001), or just before the '}' of the enclosing
// block or the end of input. Stopping there lets the caller's nextToken land
// on the next statement. If the statement already reached the '}' of its
// block, it stays on it.
//
// A '(', '[' or hash that the statement opened and never closed would hide
// every line after it, so a token that can't be inside one ends it too (see
// unclosed).
func (p *Parser) synchronize(base int) {
	p.recovering = false
	defer func() { p.blockMissing = false }()

	for !p.curTokenIs(token.EOF) && len(p.brackets) >= base {
		if len(p.brackets) > base {
			if p.unclosed(p.brackets[len(p.brackets)-1]) {
				p.closeTo(base)
				continue
			}
		} else if p.strayBrace(base) {
			p.keepOpen = true
		} else if p.blockMissing && p.peekToken.LineStart && p.peekToken.Column > p.lineIndent {
			p.openUnseen()
			p.blockMissing = false
			continue
		} else if p.statementEnded(base) {
			return
		}
		p.nextToken()
	}
}

// strayBrace reports whether peekToken is a '}' that a broken statement left
// in the middle of a line of a block that began on an earlier line. In a
// correct program it would end the block, but in a broken statement it is
// more likely a mistake, and the block goes on.
func (p *Parser) strayBrace(base int) bool {
	if base == 0 || !p.peekTokenIs(token.RBRACE) || p.peekToken.LineStart {
		return false
	}
	block := p.brackets[base-1]
	return !block.space && !block.unseen && block.tok.Line < p.peekToken.Line
}

// statementEnded reports whether a broken statement ends at curToken, when
// it has no brackets of its own open.
func (p *Parser) statementEnded(base int) bool {
	switch {
	case p.curTokenIs(token.SEMICOLON):
		return true
	case p.curTokenIs(token.RBRACE) && p.closedBlock():
		return !p.peekGoesOn()
	case p.peekTokenIs(token.EOF), base > 0 && p.peekTokenIs(token.RBRACE):
		return true
	}
	return p.peekToken.LineStart && !lexer.ContinuesLine(p.curToken.Type) && !p.peekGoesOn() && p.peekStarts()
}

// peekStarts reports whether peekToken can start a statement. A line that
// starts with anything else, like "+ 2" or ", 3", is more of the broken one.
func (p *Parser) peekStarts() bool {
	switch p.peekToken.Type {
	case token.LET, token.RETURN, token.BREAK, token.SEMICOLON, token.RBRACE:
		return true
	}
	return p.prefixParseFns[p.peekToken.Type] != nil
}

// peekGoesOn reports whether peekToken continues a statement after its '}':
// an else, an elif or a catch.
func (p *Parser) peekGoesOn() bool {
	return p.peekTokenIs(token.ELSE) || p.peekTokenIs(token.ELIF) ||
		(p.peekTokenIs(token.IDENT) && p.peekToken.Literal == "catch")
}

// unclosed reports whether peekToken shows that the open bracket b was never
// closed, because it can't be inside b:
//
//   - the end of the input;
//   - a closer of another kind, like the '}' in "f(a }";
//   - in a '(', '[' or hash, a line that starts with let, return or break, or
//     one that starts as far left as the line where b was opened (unless the
//     line before ends with an operator or a comma);
//   - in a block whose '{' was forgotten, a line that starts as far left as
//     the line of the block, or a '}' further left.
//
// The lines of a real block are statements, so they never tell.
func (p *Parser) unclosed(b bracket) bool {
	next := p.peekToken
	if next.Type == token.EOF {
		return true
	}
	closer := next.Type == token.RPAREN || next.Type == token.RBRACKET || next.Type == token.RBRACE
	switch {
	case b.unseen && next.Type == token.RBRACE:
		return next.LineStart && next.Column < b.indent
	case b.unseen:
		return closer || next.LineStart && next.Column <= b.indent
	case !b.space:
		return false
	case closer:
		return closes(next.Type) != b.tok.Type
	case !next.LineStart:
		return false
	case next.Type == token.LET || next.Type == token.RETURN || next.Type == token.BREAK:
		return true
	}
	return next.Column <= b.indent && !lexer.ContinuesLine(p.curToken.Type)
}

// neverClosed reports that the innermost open bracket is never closed, when
// peekToken shows it. It is the error of a missing ')', ']' or '}': it goes
// right after curToken, where the closer is missing, and says where the
// bracket was opened.
func (p *Parser) neverClosed() bool {
	n := len(p.brackets)
	if n == 0 || !p.brackets[n-1].space || !p.unclosed(p.brackets[n-1]) ||
		p.numberTouchesName() || p.peekTokenIs(token.ILLEGAL) {
		return false
	}
	open, missing := p.brackets[n-1].tok, p.curToken
	missing.Line, missing.Column = missing.EndLine, missing.EndColumn
	p.errorAt(missing, "the %q at line %d, column %d is never closed", open.Literal, open.Line, open.Column)
	return true
}

// expectClose moves to end, the token that closes the innermost bracket. If
// it is missing, the error says where the bracket was opened when the next
// token shows that it was never closed.
func (p *Parser) expectClose(end token.TokenType) bool {
	if p.peekTokenIs(end) {
		p.nextToken()
		return true
	}
	if p.neverClosed() {
		return false
	}
	return p.expectPeek(end)
}

// expectBlockOpen moves to the '{' that opens a block. When it is missing,
// an unseen '{' is opened in its place after the error, with the indentation
// of this line, so synchronize skips the body the program meant up to its
// '}' instead of reading the body and that '}' as more mistakes. A ')' or ']'
// in its place is more likely one too many, with the '{' after it, so then
// synchronize opens the unseen '{' only if the line ends without one and the
// next line is indented more.
func (p *Parser) expectBlockOpen() bool {
	if p.expectPeek(token.LBRACE) {
		return true
	}
	if p.peekTokenIs(token.RPAREN) || p.peekTokenIs(token.RBRACKET) {
		p.blockMissing = true
		return false
	}
	p.openUnseen()
	return false
}

// openUnseen opens the '{' of a block that the program forgot, with the
// indentation of the line of curToken.
func (p *Parser) openUnseen() {
	p.brackets = append(p.brackets, bracket{tok: p.curToken, indent: p.lineIndent, unseen: true})
	p.braceDepth++
}

// open puts tok, a '(', '[' or '{', on the stack of open brackets. A '{' goes
// on it as a block; parseHashLiteral marks the ones that open a hash.
func (p *Parser) open(tok token.Token) {
	p.brackets = append(p.brackets, bracket{tok: tok, indent: p.lineIndent, space: tok.Type != token.LBRACE})
	switch tok.Type {
	case token.LBRACE:
		p.braceDepth++
	case token.LPAREN:
		p.parenDepth++
	}
}

// close takes off the stack the bracket that closer closes, and records it
// in closed. That is the innermost bracket in a correct program. A ')' or ']' of another kind, like the one in "[f(1]",
// closes the brackets of expressions inside the one it matches too, since
// one of them was left open by mistake. A '}' does that only at the start of
// a line, where it most likely ends a block, and no closer reaches past a
// block: in "{ (1] }" the ']' closes nothing, and the parser reports it.
func (p *Parser) close(closer token.TokenType) {
	p.closed = bracket{}
	if p.keepOpen {
		p.keepOpen = false
		return
	}
	for i := len(p.brackets) - 1; i >= 0; i-- {
		b := p.brackets[i]
		if b.unseen && closer == token.RBRACE || !b.unseen && closes(closer) == b.tok.Type {
			p.closed = b
			p.closeTo(i)
			return
		}
		if !b.space || closer == token.RBRACE && !p.curToken.LineStart {
			return
		}
	}
}

// closedBlock reports whether the last closer closed a block, not a hash.
func (p *Parser) closedBlock() bool {
	return p.closed.tok.Type != "" && !p.closed.space
}

// closes returns the opening token type that closer closes.
func closes(closer token.TokenType) token.TokenType {
	switch closer {
	case token.RPAREN:
		return token.LPAREN
	case token.RBRACKET:
		return token.LBRACKET
	}
	return token.LBRACE
}

// closeTo takes brackets off the stack until n are left.
func (p *Parser) closeTo(n int) {
	for len(p.brackets) > n {
		top := p.brackets[len(p.brackets)-1]
		p.brackets = p.brackets[:len(p.brackets)-1]
		switch {
		case top.unseen || top.tok.Type == token.LBRACE:
			p.braceDepth--
		case top.tok.Type == token.LPAREN:
			p.parenDepth--
		}
	}
}

// statementBase returns how many brackets were open before the statement
// that starts at curToken. If curToken is a '(', '[' or '{', it is already
// on the stack, and it belongs to the statement.
func (p *Parser) statementBase() int {
	switch p.curToken.Type {
	case token.LPAREN, token.LBRACKET, token.LBRACE:
		return len(p.brackets) - 1
	}
	return len(p.brackets)
}

// outdentedBrace reports whether peekToken is a '}' that most likely closes
// the block around the current one, because a mistake in this block broke
// its own '}': it starts its line left of the line where this block began.
func (p *Parser) outdentedBrace(base int) bool {
	next := p.peekToken
	return len(p.errors) > p.blockErrors && next.Type == token.RBRACE && next.LineStart &&
		next.Column < p.blockIndent && base > 0 && base == len(p.brackets) && !p.brackets[base-1].space
}
