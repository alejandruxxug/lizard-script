package lexer

import (
	"lizardscript/token"
	"math"
	"strconv"
	"strings"
	"unicode/utf8"
)

// commentStart opens a comment that runs to the end of the line. It looks
// like a lizard's body, and '//' can't be used because it is integer division.
const commentStart = "-~>"

// Lexer turns lizard-script source code into a stream of tokens.
type Lexer struct {
	input        string
	position     int    // current position in input (points to current char)
	readPosition int    // current reading position in input (after current char)
	ch           byte   // current char under examination
	line         int    // 1-based line of ch
	column       int    // 1-based column of ch
	endLine      int    // line where the last token ended, 0 before the first one
	source       string // name stamped on every token, "" when unnamed
	keepComments bool   // return comments as COMMENT tokens instead of skipping them
}

// New creates a Lexer for input and loads its first character.
func New(input string) *Lexer {
	l := &Lexer{input: input, line: 1}
	l.readChar()
	return l
}

// NewNamed is New for a source that has a name, like a file name. Every
// token carries the name in its Source field, so a host that loads several
// sources can tell where a token came from. The name is never part of any
// message. New(input) is NewNamed(input, "").
func NewNamed(input, name string) *Lexer {
	l := New(input)
	l.source = name
	return l
}

// KeepComments makes the lexer return each -~> comment as a COMMENT token,
// whose literal is the whole comment, instead of skipping it. It is for
// tools that show the source, like a syntax highlighter; the parser must not
// be given such a lexer.
func (l *Lexer) KeepComments() {
	l.keepComments = true
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
	if !isContinuationByte(l.ch) {
		l.column += 1
	}
}

// atEnd reports whether the whole input was read. ch alone can't tell, since
// the source could contain a real NUL character.
func (l *Lexer) atEnd() bool {
	return l.position >= len(l.input)
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
// the line and column where it starts and ends and whether it starts its
// line.
func (l *Lexer) NextToken() token.Token {
	l.skipWhitespace()

	line, column := l.line, l.column
	var tok token.Token
	if l.keepComments && l.atComment() {
		tok = l.readComment()
	} else {
		tok = l.readToken()
	}
	tok.Line, tok.Column = line, column
	tok.EndLine, tok.EndColumn = l.line, l.column
	tok.LineStart = line > l.endLine
	tok.Source = l.source
	l.endLine = l.line
	return tok
}

// ContinuesLine reports whether a line that ends with a token of type t goes
// on to the next line: t is a binary operator, an assignment, a '!', a '.',
// a ',' or an opening bracket. Every other line ends its statement, unless
// it is inside an open '(', '[' or hash. The parser and the REPL both follow
// this rule.
func ContinuesLine(t token.TokenType) bool {
	switch t {
	case token.PLUS, token.MINUS, token.ASTERISK, token.SLASH, token.DOUBLE_SLASH,
		token.POW, token.MODULE, token.EQ, token.NOT_EQ, token.EXACT, token.NOT_EXACT,
		token.LT, token.GT, token.LTE, token.GTE, token.AND, token.OR,
		token.ASSIGN, token.PLUS_ASSIGN, token.MINUS_ASSIGN, token.ASTERISK_ASSIGN, token.SLASH_ASSIGN,
		token.BANG, token.DOT, token.COMMA, token.LPAREN, token.LBRACKET, token.LBRACE:
		return true
	}
	return false
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
		if tok.Type == token.ASTERISK {
			tok = l.extendToken(tok, '=', token.ASTERISK_ASSIGN)
		}
	case '+':
		tok = l.oneOrTwoCharToken('=', token.PLUS, token.PLUS_ASSIGN)
	case '-':
		tok = l.oneOrTwoCharToken('=', token.MINUS, token.MINUS_ASSIGN)
	case '/':
		tok = l.oneOrTwoCharToken('/', token.SLASH, token.DOUBLE_SLASH)
		if tok.Type == token.SLASH {
			tok = l.extendToken(tok, '=', token.SLASH_ASSIGN)
		}
	case '%':
		tok = newToken(token.MODULE, l.ch)
	case '&':
		tok = l.oneOrTwoCharToken('&', token.ILLEGAL, token.AND)
	case '|':
		tok = l.oneOrTwoCharToken('|', token.ILLEGAL, token.OR)
	case '"':
		tok = l.readString()
	case '.':
		tok = newToken(token.DOT, l.ch)
	case '#':
		return l.readColor()
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
		if !l.atEnd() {
			return l.readIllegal()
		}
		tok.Literal = ""
		tok.Type = token.EOF
	default:
		if isLetter(l.ch) {
			tok.Literal = l.readIdentifier()
			tok.Type = token.LookupIdent(tok.Literal)
			return tok
		} else if isDigit(l.ch) {
			return l.readNumber()
		}
		return l.readIllegal()
	}

	l.readChar()
	return tok
}

// readIllegal returns an ILLEGAL token with the whole character at the
// current position. A character like 'ñ' takes several bytes in UTF-8, so
// reading one byte would report a broken piece of it.
func (l *Lexer) readIllegal() token.Token {
	_, size := utf8.DecodeRuneInString(l.input[l.position:])
	literal := l.input[l.position : l.position+size]
	for range size {
		l.readChar()
	}
	return token.Token{Type: token.ILLEGAL, Literal: literal}
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
		case !l.keepComments && l.atComment():
			l.skipComment()
		default:
			return
		}
	}
}

// atComment reports whether a comment starts at the current character.
func (l *Lexer) atComment() bool {
	return strings.HasPrefix(l.input[l.position:], commentStart)
}

// readComment reads a comment up to the end of its line, leaving the
// newline (or the "\r" of a "\r\n") in ch, so it is not part of the comment.
func (l *Lexer) readComment() token.Token {
	start := l.position
	for l.ch != '\n' && !l.atEnd() && !(l.ch == '\r' && l.peekChar() == '\n') {
		l.readChar()
	}
	return token.Token{Type: token.COMMENT, Literal: l.input[start:l.position]}
}

// skipComment skips up to the end of the line, leaving the newline in ch.
func (l *Lexer) skipComment() {
	for l.ch != '\n' && !l.atEnd() {
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
// The literal is the content without quotes, with the escapes \" \\ \n and
// \t already replaced by the character they mean. If the input ends first, it
// returns an ILLEGAL token whose literal keeps the opening quote. An unknown
// escape returns an ILLEGAL token whose literal is the backslash and the
// character after it, like \q.
func (l *Lexer) readString() token.Token {
	start := l.position
	var content strings.Builder
	for {
		l.readChar()
		if l.atEnd() {
			return token.Token{Type: token.ILLEGAL, Literal: l.input[start:l.position]}
		}
		switch l.ch {
		case '"':
			return token.Token{Type: token.STRING, Literal: content.String()}
		case '\\':
			l.readChar()
			if l.atEnd() {
				return token.Token{Type: token.ILLEGAL, Literal: l.input[start:l.position]}
			}
			if l.ch == 'u' {
				if bad, ok := l.readUnicodeEscape(&content); !ok {
					return bad
				}
				continue
			}
			escaped, ok := unescape(l.ch)
			if !ok {
				return l.unknownEscape()
			}
			content.WriteByte(escaped)
		default:
			content.WriteByte(l.ch)
		}
	}
}

// unescape returns the character that the escape \ch stands for.
func unescape(ch byte) (byte, bool) {
	switch ch {
	case '"', '\\':
		return ch, true
	case 'n':
		return '\n', true
	case 't':
		return '\t', true
	case 'r':
		return '\r', true
	}
	return 0, false
}

// readUnicodeEscape reads the rest of \uXXXX (exactly 4 hex digits) or
// \u{X...} (1 to 6 hex digits), with ch on the 'u', and writes the code
// point to content. It leaves ch on the last character of the escape. A
// bad escape (too few digits, a missing }, a surrogate half, a value above
// 10FFFF) returns an ILLEGAL token whose literal is the escape as written;
// the parser explains it with ExplainUnicodeEscape. The rest of the string
// is skipped, like for an unknown escape.
func (l *Lexer) readUnicodeEscape(content *strings.Builder) (token.Token, bool) {
	start := l.position - 1 // the backslash
	braces := l.peekChar() == '{'
	if braces {
		l.readChar()
	}
	digits := 0
	for isHexDigit(l.peekChar()) && digits < 6 && (braces || digits < 4) {
		l.readChar()
		digits++
	}
	if braces && l.peekChar() == '}' {
		l.readChar()
	}
	literal := l.input[start : l.position+1]
	if ExplainUnicodeEscape(literal) != "" {
		l.skipRestOfString()
		return token.Token{Type: token.ILLEGAL, Literal: literal}, false
	}
	value, _ := strconv.ParseUint(strings.Trim(literal[2:], "{}"), 16, 32)
	content.WriteRune(rune(value))
	return token.Token{}, true
}

// ExplainUnicodeEscape says what is wrong with the escape text (like \u12
// or \u{D800}) that readUnicodeEscape rejected, or returns "" when the
// escape is fine.
func ExplainUnicodeEscape(escape string) string {
	body := escape[2:]
	if strings.HasPrefix(body, "{") {
		if !strings.HasSuffix(body, "}") || len(body) < 3 {
			return "invalid unicode escape " + escape + ": \\u{...} needs 1 to 6 hex digits and a closing }"
		}
		body = body[1 : len(body)-1]
	} else if len(body) != 4 {
		return "invalid unicode escape " + escape + ": \\u needs exactly 4 hex digits"
	}
	value, _ := strconv.ParseUint(body, 16, 32)
	switch {
	case value >= 0xD800 && value <= 0xDFFF:
		return "invalid unicode escape " + escape + ": surrogate halves (D800 to DFFF) are not characters"
	case value > 0x10FFFF:
		return "invalid unicode escape " + escape + ": code point above 10FFFF"
	}
	return ""
}

// skipRestOfString moves ch to the closing quote of the string (or the end
// of input), so that quote is not read as the start of another string.
func (l *Lexer) skipRestOfString() {
	for !l.atEnd() && l.ch != '"' {
		if l.ch == '\\' {
			l.readChar()
		}
		l.readChar()
	}
}

// unknownEscape returns the ILLEGAL token for an escape like \q, with ch on
// the character after the backslash. It then skips the rest of the string, so
// its closing quote is not read as the start of another one.
func (l *Lexer) unknownEscape() token.Token {
	_, size := utf8.DecodeRuneInString(l.input[l.position:])
	literal := "\\" + l.input[l.position:l.position+size]

	for !l.atEnd() && l.ch != '"' {
		if l.ch == '\\' {
			l.readChar()
		}
		l.readChar()
	}
	return token.Token{Type: token.ILLEGAL, Literal: literal}
}

// readColor reads a color: '#' and then 3, 6 or 8 hex digits. It takes the
// whole word after the '#', so #12 and #12xyz are one ILLEGAL token whose
// literal is the text as written, instead of a color followed by a name. A
// '#' alone is an ILLEGAL character, as it always was.
func (l *Lexer) readColor() token.Token {
	start := l.position
	l.readChar()
	for isLetter(l.ch) || isDigit(l.ch) {
		l.readChar()
	}
	text := l.input[start:l.position]

	digits := text[1:]
	if len(digits) != 3 && len(digits) != 6 && len(digits) != 8 {
		return token.Token{Type: token.ILLEGAL, Literal: text}
	}
	for i := 0; i < len(digits); i++ {
		if !isHexDigit(digits[i]) {
			return token.Token{Type: token.ILLEGAL, Literal: text}
		}
	}
	return token.Token{Type: token.COLOR, Literal: text}
}

func isHexDigit(ch byte) bool {
	return isDigit(ch) || 'a' <= ch && ch <= 'f' || 'A' <= ch && ch <= 'F'
}

// readNumber reads a number: digits, optionally a '.' and more digits,
// optionally an exponent (1e3, 2.5e-4, 1E+6), and optionally a unit right
// after it. A '.' that isn't followed by a digit is not part of the number,
// so 1.x is the integer 1, a dot and a name. The same goes for an 'e' that
// isn't followed by digits, so 1else is the integer 1 and a name.
func (l *Lexer) readNumber() token.Token {
	start := l.position
	tokenType := token.TokenType(token.INT)
	l.skipDigits()
	if l.ch == '.' && isDigit(l.peekChar()) {
		tokenType = token.FLOAT
		l.readChar()
		l.skipDigits()
	}
	if l.startsExponent() {
		tokenType = token.FLOAT
		l.readChar()
		if l.ch == '+' || l.ch == '-' {
			l.readChar()
		}
		l.skipDigits()
	}
	number := l.input[start:l.position]

	if !isLetter(l.ch) {
		return token.Token{Type: tokenType, Literal: number}
	}
	return l.readUnit(number, tokenType)
}

// startsExponent reports whether ch is the 'e' or 'E' of an exponent: one
// that is followed by digits, or by a sign and digits.
func (l *Lexer) startsExponent() bool {
	if l.ch != 'e' && l.ch != 'E' {
		return false
	}
	next := l.peekChar()
	if next == '+' || next == '-' {
		after := l.position + 2
		return after < len(l.input) && isDigit(l.input[after])
	}
	return isDigit(next)
}

func (l *Lexer) skipDigits() {
	for isDigit(l.ch) {
		l.readChar()
	}
}

// readUnit reads the unit that follows number, and returns the number
// converted to it. deg, rad and turn give a FLOAT in radians, and px keeps
// the number as it is. The unit is only taken when the whole word is one of
// them: after any other word (5kg, 5degx) the number is left alone and the
// word is read next as an identifier, as it always was.
func (l *Lexer) readUnit(number string, tokenType token.TokenType) token.Token {
	end := l.position
	for end < len(l.input) && (isLetter(l.input[end]) || isDigit(l.input[end])) {
		end++
	}
	unit := l.input[l.position:end]
	value, _ := strconv.ParseFloat(number, 64)

	switch unit {
	case "px":
	case "deg":
		value = value * math.Pi / 180
	case "rad":
	case "turn":
		value = value * 2 * math.Pi
	default:
		return token.Token{Type: tokenType, Literal: number}
	}
	l.readIdentifier()
	if unit == "px" {
		return token.Token{Type: tokenType, Literal: number}
	}

	// A number too big for a float turns into +Inf, which the parser rejects.
	literal := strconv.FormatFloat(value, 'g', -1, 64)
	if !strings.ContainsAny(literal, ".eIN") {
		literal += ".0"
	}
	return token.Token{Type: token.FLOAT, Literal: literal}
}

func isLetter(ch byte) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_'
}

func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}

// isContinuationByte reports whether ch is the second or later byte of a
// UTF-8 character, which doesn't take a new column.
func isContinuationByte(ch byte) bool {
	return ch&0xC0 == 0x80
}

func newToken(tokenType token.TokenType, ch byte) token.Token {
	return token.Token{Type: tokenType, Literal: string(ch)}
}
