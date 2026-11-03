package token

// TokenType identifies what kind of token the lexer produced.
type TokenType string

// Token is a single lexical unit of lizard-script source code.
type Token struct {
	Type    TokenType
	Literal string
	Line    int // 1-based line where the token starts
	Column  int // 1-based column where the token starts
	// EndLine and EndColumn are where the token ends: the position just
	// after its last character, counted like Line and Column. A string
	// that spans lines ends on a later line than it starts.
	EndLine   int
	EndColumn int
	// LineStart is true when the token is the first one on its line: a line
	// break (or the start of the input) comes before it. A string that spans
	// lines doesn't count, since its line break is inside the token.
	LineStart bool
	// Source is the name of the source the token was read from, given to
	// lexer.NewNamed. It is "" for an unnamed source. It never appears in
	// messages; it is for hosts that load several sources.
	Source string
}

const (
	ILLEGAL = "ILLEGAL"
	EOF     = "EOF"

	// COMMENT is a -~> comment. The lexer only returns it when asked to
	// (Lexer.KeepComments), for tools like a syntax highlighter; the parser
	// never sees one.
	COMMENT = "COMMENT"

	// Identifiers + literals
	IDENT  = "IDENT"  // add, foobar, x, y, ...
	INT    = "INT"    // 1343456
	FLOAT  = "FLOAT"  // 1.5
	STRING = "STRING" // "foobar"
	COLOR  = "COLOR"  // #3fa34d

	// Operators
	ASSIGN       = "="
	PLUS         = "+"
	MINUS        = "-"
	BANG         = "!"
	ASTERISK     = "*"
	SLASH        = "/"
	DOUBLE_SLASH = "//"
	POW          = "**"
	MODULE       = "%"
	EXACT        = "==="
	NOT_EXACT    = "!=="

	// Compound assignment: x += 1 changes x by the operator before the '='.
	PLUS_ASSIGN     = "+="
	MINUS_ASSIGN    = "-="
	ASTERISK_ASSIGN = "*="
	SLASH_ASSIGN    = "/="

	LT  = "<"
	GT  = ">"
	LTE = "<="
	GTE = ">="

	EQ     = "=="
	NOT_EQ = "!="

	AND = "&&"
	OR  = "||"

	// Delimiters
	COMMA     = ","
	SEMICOLON = ";"
	COLON     = ":"
	DOT       = "."

	LPAREN   = "("
	RPAREN   = ")"
	LBRACE   = "{"
	RBRACE   = "}"
	LBRACKET = "["
	RBRACKET = "]"

	// Keywords
	FUNCTION = "FUNCTION"
	LET      = "LET"
	TRUE     = "TRUE"
	FALSE    = "FALSE"
	IF       = "IF"
	ELSE     = "ELSE"
	ELIF     = "ELIF"
	RETURN   = "RETURN"
	FOR      = "FOR"
	WHILE    = "WHILE"
	BREAK    = "BREAK"
)

var keywords = map[string]TokenType{
	"function": FUNCTION,
	"let":      LET,
	"true":     TRUE,
	"false":    FALSE,
	"if":       IF,
	"else":     ELSE,
	"elif":     ELIF,
	"return":   RETURN,
	"for":      FOR,
	"while":    WHILE,
	"break":    BREAK,
}

// LookupIdent returns the keyword token type for ident, or IDENT if it
// isn't a reserved word.
func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}
