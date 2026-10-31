package token

// TokenType identifies what kind of token the lexer produced.
type TokenType string

// Token is a single lexical unit of lizard-script source code.
type Token struct {
	Type    TokenType
	Literal string
	Line    int // 1-based line where the token starts
	Column  int // 1-based column where the token starts
}

const (
	ILLEGAL = "ILLEGAL"
	EOF     = "EOF"

	// Identifiers + literals
	IDENT  = "IDENT"  // add, foobar, x, y, ...
	INT    = "INT"    // 1343456
	STRING = "STRING" // "foobar"

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
