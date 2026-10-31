package lexer

import "lizardscript/token"

// Lexer turns lizard-script source code into a stream of tokens.
type Lexer struct {
	input        string
	position     int  // current position in input (points to current char)
	readPosition int  // current reading position in input (after current char)
	ch           byte // current char under examination
}

// New creates a Lexer for input and loads its first character.
func New(input string) *Lexer {
	l := &Lexer{input: input}
	l.readChar()
	return l
}

// readChar advances one character. At the end of input ch becomes 0 (NUL).
func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.readPosition]
	}
	l.position = l.readPosition
	l.readPosition += 1
}

// peekChar returns the next character without advancing the lexer.
// Used for two-character operators like '==' or '!='.
func (l *Lexer) peekChar() byte {
	if l.readPosition >= len(l.input) {
		return 0
	}
	return l.input[l.readPosition]
}

// NextToken reads and returns the next token from the input.
func (l *Lexer) NextToken() token.Token {
	var tok token.Token

	l.skipWhitespace()

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
	case ';':
		tok = newToken(token.SEMICOLON, l.ch)
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

// skipWhitespace skips spaces, tabs and newlines.
func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
		l.readChar()
	}
}

// readIdentifier reads letters (and underscores) and returns the whole word.
func (l *Lexer) readIdentifier() string {
	position := l.position
	for isLetter(l.ch) {
		l.readChar()
	}
	return l.input[position:l.position]
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
