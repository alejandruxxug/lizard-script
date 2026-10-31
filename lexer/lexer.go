package lexer

import (
	"lizardscript/token"
	"strings"
)

// commentStart opens a comment that runs to the end of the line. It looks
// like a lizard's body, and '//' can't be used because it is integer division.
const commentStart = "-~>"

// Lexer turns lizard-script source code into a stream of tokens.
type Lexer struct {
	input        string
	position     int  // current position in input (points to current char)
	readPosition int  // current reading position in input (after current char)
	ch           byte // current char under examination
	line         int  // 1-based line of ch
	column       int  // 1-based column of ch
}

// New creates a Lexer for input and loads its first character.
func New(input string) *Lexer {
	l := &Lexer{input: input, line: 1}
	l.readChar()
	return l
}

// readChar advances one character. At the end of input ch becomes 0 (NUL)
// and further calls do nothing, so the EOF position stays the same.
func (l *Lexer) readChar() {
	if l.readPosition > len(l.input) {
		return
	}
	if l.ch == '\n' {
		l.line += 1
		l.column = 0
	}
	if l.readPosition == len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.readPosition]
	}
	l.position = l.readPosition
	l.readPosition += 1
	l.column += 1
}

// peekChar returns the next character without advancing the lexer.
// Used for two-character operators like '==' or '!='.
func (l *Lexer) peekChar() byte {
	if l.readPosition >= len(l.input) {
		return 0
	}
	return l.input[l.readPosition]
}

// NextToken reads and returns the next token from the input, stamped with
// the line and column where it starts.
func (l *Lexer) NextToken() token.Token {
	l.skipWhitespace()

	line, column := l.line, l.column
	tok := l.readToken()
	tok.Line, tok.Column = line, column
	return tok
}

// readToken reads the token that starts at the current character.
func (l *Lexer) readToken() token.Token {
	var tok token.Token

	switch l.ch {
	case '=':
		tok = l.oneOrTwoCharToken('=', token.ASSIGN, token.EQ)
		if tok.Type == token.EQ {
			tok = l.extendToken(tok, '=', token.EXACT)
		}
	case '!':
		tok = l.oneOrTwoCharToken('=', token.BANG, token.NOT_EQ)
		if tok.Type == token.NOT_EQ {
			tok = l.extendToken(tok, '=', token.NOT_EXACT)
		}
	case '<':
		tok = l.oneOrTwoCharToken('=', token.LT, token.LTE)
	case '>':
		tok = l.oneOrTwoCharToken('=', token.GT, token.GTE)
	case '*':
		tok = l.oneOrTwoCharToken('*', token.ASTERISK, token.POW)
	case '+':
		tok = newToken(token.PLUS, l.ch)
	case '-':
		tok = newToken(token.MINUS, l.ch)
	case '/':
		tok = l.oneOrTwoCharToken('/', token.SLASH, token.DOUBLE_SLASH)
	case '%':
		tok = newToken(token.MODULE, l.ch)
	case '&':
		tok = l.oneOrTwoCharToken('&', token.ILLEGAL, token.AND)
	case '|':
		tok = l.oneOrTwoCharToken('|', token.ILLEGAL, token.OR)
	case '"':
		tok = l.readString()
	case ';':
		tok = newToken(token.SEMICOLON, l.ch)
	case ':':
		tok = newToken(token.COLON, l.ch)
	case ',':
		tok = newToken(token.COMMA, l.ch)
	case '(':
		tok = newToken(token.LPAREN, l.ch)
	case ')':
		tok = newToken(token.RPAREN, l.ch)
	case '{':
		tok = newToken(token.LBRACE, l.ch)
	case '}':
		tok = newToken(token.RBRACE, l.ch)
	case '[':
		tok = newToken(token.LBRACKET, l.ch)
	case ']':
		tok = newToken(token.RBRACKET, l.ch)
	case 0:
		tok.Literal = ""
		tok.Type = token.EOF
	default:
		if isLetter(l.ch) {
			tok.Literal = l.readIdentifier()
			tok.Type = token.LookupIdent(tok.Literal)
			return tok
		} else if isDigit(l.ch) {
			tok.Type = token.INT
			tok.Literal = l.readNumber()
			return tok
		}
		tok = newToken(token.ILLEGAL, l.ch)
	}

	l.readChar()
	return tok
}

// oneOrTwoCharToken returns a two-character token of type double when the
// next char is second (e.g. '<' followed by '='), otherwise a one-character
// token of type single.
func (l *Lexer) oneOrTwoCharToken(second byte, single, double token.TokenType) token.Token {
	if l.peekChar() == second {
		first := l.ch
		l.readChar()
		return token.Token{Type: double, Literal: string(first) + string(l.ch)}
	}
	return newToken(single, l.ch)
}

// extendToken appends the next char to tok and turns it into tokenType when
// that char is next (e.g. '==' followed by '=' becomes '==='), otherwise it
// returns tok unchanged.
func (l *Lexer) extendToken(tok token.Token, next byte, tokenType token.TokenType) token.Token {
	if l.peekChar() == next {
		l.readChar()
		return token.Token{Type: tokenType, Literal: tok.Literal + string(l.ch)}
	}
	return tok
}

// skipWhitespace skips spaces, tabs, newlines and comments.
func (l *Lexer) skipWhitespace() {
	for {
		switch {
		case l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r':
			l.readChar()
		case strings.HasPrefix(l.input[l.position:], commentStart):
			l.skipComment()
		default:
			return
		}
	}
}

// skipComment skips up to the end of the line, leaving the newline in ch.
func (l *Lexer) skipComment() {
	for l.ch != '\n' && l.ch != 0 {
		l.readChar()
	}
}

// readIdentifier reads a word and returns it. A word starts with a letter or
// underscore (checked by the caller) and can continue with digits, like x1.
func (l *Lexer) readIdentifier() string {
	position := l.position
	for isLetter(l.ch) || isDigit(l.ch) {
		l.readChar()
	}
	return l.input[position:l.position]
}

// readString reads a double-quoted string, leaving ch on the closing quote.
// The literal is the content without quotes. If the input ends first, it
// returns an ILLEGAL token whose literal keeps the opening quote.
func (l *Lexer) readString() token.Token {
	position := l.position + 1
	for {
		l.readChar()
		if l.ch == '"' {
			return token.Token{Type: token.STRING, Literal: l.input[position:l.position]}
		}
		if l.ch == 0 {
			return token.Token{Type: token.ILLEGAL, Literal: l.input[position-1 : l.position]}
		}
	}
}

// readNumber reads consecutive digits and returns them as a string.
func (l *Lexer) readNumber() string {
	position := l.position
	for isDigit(l.ch) {
		l.readChar()
	}
	return l.input[position:l.position]
}

func isLetter(ch byte) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_'
}

func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}

func newToken(tokenType token.TokenType, ch byte) token.Token {
	return token.Token{Type: tokenType, Literal: string(ch)}
}
