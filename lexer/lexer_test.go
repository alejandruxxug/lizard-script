package lexer

import (
	"lizardscript/token"
	"strings"
	"testing"
)

func TestNextToken(t *testing.T) {
	input := `let five = 5;
let ten = 10;

function add(x, y) {
  return x + y;
}

let result = add(five, ten);
!-/*5;
5 < 10 > 5;

if (5 < 10) {
	return true;
} elif (5 > 10) {
	return false;
} else {
	break;
}

for while

10 == 10;
10 != 9;
10 <= 10;
10 >= 9;
2 ** 3;
10 // 3;
10 % 3;
10 === 10;
10 !== 9;
[1, 2];
@
`

	tests := []struct {
		expectedType    token.TokenType
		expectedLiteral string
	}{
		{token.LET, "let"},
		{token.IDENT, "five"},
		{token.ASSIGN, "="},
		{token.INT, "5"},
		{token.SEMICOLON, ";"},
		{token.LET, "let"},
		{token.IDENT, "ten"},
		{token.ASSIGN, "="},
		{token.INT, "10"},
		{token.SEMICOLON, ";"},
		{token.FUNCTION, "function"},
		{token.IDENT, "add"},
		{token.LPAREN, "("},
		{token.IDENT, "x"},
		{token.COMMA, ","},
		{token.IDENT, "y"},
		{token.RPAREN, ")"},
		{token.LBRACE, "{"},
		{token.RETURN, "return"},
		{token.IDENT, "x"},
		{token.PLUS, "+"},
		{token.IDENT, "y"},
		{token.SEMICOLON, ";"},
		{token.RBRACE, "}"},
		{token.LET, "let"},
		{token.IDENT, "result"},
		{token.ASSIGN, "="},
		{token.IDENT, "add"},
		{token.LPAREN, "("},
		{token.IDENT, "five"},
		{token.COMMA, ","},
		{token.IDENT, "ten"},
		{token.RPAREN, ")"},
		{token.SEMICOLON, ";"},
		{token.BANG, "!"},
		{token.MINUS, "-"},
		{token.SLASH, "/"},
		{token.ASTERISK, "*"},
		{token.INT, "5"},
		{token.SEMICOLON, ";"},
		{token.INT, "5"},
		{token.LT, "<"},
		{token.INT, "10"},
		{token.GT, ">"},
		{token.INT, "5"},
		{token.SEMICOLON, ";"},
		{token.IF, "if"},
		{token.LPAREN, "("},
		{token.INT, "5"},
		{token.LT, "<"},
		{token.INT, "10"},
		{token.RPAREN, ")"},
		{token.LBRACE, "{"},
		{token.RETURN, "return"},
		{token.TRUE, "true"},
		{token.SEMICOLON, ";"},
		{token.RBRACE, "}"},
		{token.ELIF, "elif"},
		{token.LPAREN, "("},
		{token.INT, "5"},
		{token.GT, ">"},
		{token.INT, "10"},
		{token.RPAREN, ")"},
		{token.LBRACE, "{"},
		{token.RETURN, "return"},
		{token.FALSE, "false"},
		{token.SEMICOLON, ";"},
		{token.RBRACE, "}"},
		{token.ELSE, "else"},
		{token.LBRACE, "{"},
		{token.BREAK, "break"},
		{token.SEMICOLON, ";"},
		{token.RBRACE, "}"},
		{token.FOR, "for"},
		{token.WHILE, "while"},
		{token.INT, "10"},
		{token.EQ, "=="},
		{token.INT, "10"},
		{token.SEMICOLON, ";"},
		{token.INT, "10"},
		{token.NOT_EQ, "!="},
		{token.INT, "9"},
		{token.SEMICOLON, ";"},
		{token.INT, "10"},
		{token.LTE, "<="},
		{token.INT, "10"},
		{token.SEMICOLON, ";"},
		{token.INT, "10"},
		{token.GTE, ">="},
		{token.INT, "9"},
		{token.SEMICOLON, ";"},
		{token.INT, "2"},
		{token.POW, "**"},
		{token.INT, "3"},
		{token.SEMICOLON, ";"},
		{token.INT, "10"},
		{token.DOUBLE_SLASH, "//"},
		{token.INT, "3"},
		{token.SEMICOLON, ";"},
		{token.INT, "10"},
		{token.MODULE, "%"},
		{token.INT, "3"},
		{token.SEMICOLON, ";"},
		{token.INT, "10"},
		{token.EXACT, "==="},
		{token.INT, "10"},
		{token.SEMICOLON, ";"},
		{token.INT, "10"},
		{token.NOT_EXACT, "!=="},
		{token.INT, "9"},
		{token.SEMICOLON, ";"},
		{token.LBRACKET, "["},
		{token.INT, "1"},
		{token.COMMA, ","},
		{token.INT, "2"},
		{token.RBRACKET, "]"},
		{token.SEMICOLON, ";"},
		{token.ILLEGAL, "@"},
		{token.EOF, ""},
	}

	l := New(input)

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

// TestOperatorEdgeCases checks that the lexer always takes the longest
// operator it can, and falls back to shorter ones otherwise.
func TestOperatorEdgeCases(t *testing.T) {
	tests := []struct {
		input    string
		expected []token.Token
	}{
		{"====", []token.Token{{Type: token.EXACT, Literal: "==="}, {Type: token.ASSIGN, Literal: "="}}},
		{"!===", []token.Token{{Type: token.NOT_EXACT, Literal: "!=="}, {Type: token.ASSIGN, Literal: "="}}},
		{"== =", []token.Token{{Type: token.EQ, Literal: "=="}, {Type: token.ASSIGN, Literal: "="}}},
		{"!!=", []token.Token{{Type: token.BANG, Literal: "!"}, {Type: token.NOT_EQ, Literal: "!="}}},
		{"///", []token.Token{{Type: token.DOUBLE_SLASH, Literal: "//"}, {Type: token.SLASH, Literal: "/"}}},
		{"***", []token.Token{{Type: token.POW, Literal: "**"}, {Type: token.ASTERISK, Literal: "*"}}},
		{"a%b", []token.Token{{Type: token.IDENT, Literal: "a"}, {Type: token.MODULE, Literal: "%"}, {Type: token.IDENT, Literal: "b"}}},
		{"x===y", []token.Token{{Type: token.IDENT, Literal: "x"}, {Type: token.EXACT, Literal: "==="}, {Type: token.IDENT, Literal: "y"}}},
		{"==", []token.Token{{Type: token.EQ, Literal: "=="}}}, // ends right after '==' (peek hits EOF)
	}

	for _, tt := range tests {
		l := New(tt.input)
		for i, want := range append(tt.expected, token.Token{Type: token.EOF, Literal: ""}) {
			got := l.NextToken()
			if got.Type != want.Type || got.Literal != want.Literal {
				t.Fatalf("input %q, token[%d] - expected=%+v, got=%+v",
					tt.input, i, want, got)
			}
		}
	}
}

// TestNewTokens checks strings, logical operators, colons and identifiers
// with digits.
func TestNewTokens(t *testing.T) {
	tests := []struct {
		input    string
		expected []token.Token
	}{
		{`"hello world"`, []token.Token{{Type: token.STRING, Literal: "hello world"}}},
		{`""`, []token.Token{{Type: token.STRING, Literal: ""}}},
		{`"open`, []token.Token{{Type: token.ILLEGAL, Literal: `"open`}}},
		{"a && b", []token.Token{{Type: token.IDENT, Literal: "a"}, {Type: token.AND, Literal: "&&"}, {Type: token.IDENT, Literal: "b"}}},
		{"a || b", []token.Token{{Type: token.IDENT, Literal: "a"}, {Type: token.OR, Literal: "||"}, {Type: token.IDENT, Literal: "b"}}},
		{"&", []token.Token{{Type: token.ILLEGAL, Literal: "&"}}},
		{"|", []token.Token{{Type: token.ILLEGAL, Literal: "|"}}},
		{`{"a": 1}`, []token.Token{
			{Type: token.LBRACE, Literal: "{"},
			{Type: token.STRING, Literal: "a"},
			{Type: token.COLON, Literal: ":"},
			{Type: token.INT, Literal: "1"},
			{Type: token.RBRACE, Literal: "}"},
		}},
		{"x1 _a2b", []token.Token{{Type: token.IDENT, Literal: "x1"}, {Type: token.IDENT, Literal: "_a2b"}}},
		{"1x", []token.Token{{Type: token.INT, Literal: "1"}, {Type: token.IDENT, Literal: "x"}}},
	}

	for _, tt := range tests {
		l := New(tt.input)
		for i, want := range append(tt.expected, token.Token{Type: token.EOF, Literal: ""}) {
			got := l.NextToken()
			if got.Type != want.Type || got.Literal != want.Literal {
				t.Fatalf("input %q, token[%d] - expected=%+v, got=%+v",
					tt.input, i, want, got)
			}
		}
	}
}

// TestComments checks that -~> comments are skipped and that it only opens a
// comment when all three characters are together.
func TestComments(t *testing.T) {
	tests := []struct {
		input    string
		expected []token.Token
	}{
		{"1; -~> trailing\n2", []token.Token{{Type: token.INT, Literal: "1"}, {Type: token.SEMICOLON, Literal: ";"}, {Type: token.INT, Literal: "2"}}},
		{"-~> whole line\n1", []token.Token{{Type: token.INT, Literal: "1"}}},
		{"-~> one\n-~> two\n\n  -~> three\n1", []token.Token{{Type: token.INT, Literal: "1"}}},
		{"1 -~> no newline at the end", []token.Token{{Type: token.INT, Literal: "1"}}},
		{"-~>", nil},
		{`"-~> text"`, []token.Token{{Type: token.STRING, Literal: "-~> text"}}},
		{"a -~", []token.Token{{Type: token.IDENT, Literal: "a"}, {Type: token.MINUS, Literal: "-"}, {Type: token.ILLEGAL, Literal: "~"}}},
		{"~", []token.Token{{Type: token.ILLEGAL, Literal: "~"}}},
		{"a -b", []token.Token{{Type: token.IDENT, Literal: "a"}, {Type: token.MINUS, Literal: "-"}, {Type: token.IDENT, Literal: "b"}}},
		{"a - > b", []token.Token{{Type: token.IDENT, Literal: "a"}, {Type: token.MINUS, Literal: "-"}, {Type: token.GT, Literal: ">"}, {Type: token.IDENT, Literal: "b"}}},
	}

	for _, tt := range tests {
		l := New(tt.input)
		for i, want := range append(tt.expected, token.Token{Type: token.EOF, Literal: ""}) {
			got := l.NextToken()
			if got.Type != want.Type || got.Literal != want.Literal {
				t.Fatalf("input %q, token[%d] - expected=%+v, got=%+v",
					tt.input, i, want, got)
			}
		}
	}
}

// TestNonASCIIAndNUL checks that a multi-byte character is one ILLEGAL token
// and not several pieces, and that a NUL in the source doesn't end the input.
func TestNonASCIIAndNUL(t *testing.T) {
	tests := []struct {
		input    string
		expected []token.Token
	}{
		{"añ", []token.Token{{Type: token.IDENT, Literal: "a"}, {Type: token.ILLEGAL, Literal: "ñ"}}},
		{"🦎 x", []token.Token{{Type: token.ILLEGAL, Literal: "🦎"}, {Type: token.IDENT, Literal: "x"}}},
		{"\xff", []token.Token{{Type: token.ILLEGAL, Literal: "\xff"}}},
		{`"ñandú 🦎"`, []token.Token{{Type: token.STRING, Literal: "ñandú 🦎"}}},
		{"-~> ñandú 🦎\n1", []token.Token{{Type: token.INT, Literal: "1"}}},
		{"a\x00b", []token.Token{{Type: token.IDENT, Literal: "a"}, {Type: token.ILLEGAL, Literal: "\x00"}, {Type: token.IDENT, Literal: "b"}}},
		{"\"a\x00b\"", []token.Token{{Type: token.STRING, Literal: "a\x00b"}}},
		{"-~> a\x00b\n1", []token.Token{{Type: token.INT, Literal: "1"}}},
	}

	for _, tt := range tests {
		l := New(tt.input)
		for i, want := range append(tt.expected, token.Token{Type: token.EOF, Literal: ""}) {
			got := l.NextToken()
			if got.Type != want.Type || got.Literal != want.Literal {
				t.Fatalf("input %q, token[%d] - expected=%+v, got=%+v",
					tt.input, i, want, got)
			}
		}
	}
}

// TestColumnsCountCharacters checks that columns count characters and not
// bytes, so an accent doesn't shift the rest of the line.
func TestColumnsCountCharacters(t *testing.T) {
	l := New("\"canción 🦎\" x\ny")

	if tok := l.NextToken(); tok.Column != 1 {
		t.Fatalf("string - expected column 1, got %d", tok.Column)
	}
	if tok := l.NextToken(); tok.Literal != "x" || tok.Column != 13 {
		t.Fatalf("expected x at column 13, got %q at %d", tok.Literal, tok.Column)
	}
	if tok := l.NextToken(); tok.Literal != "y" || tok.Line != 2 || tok.Column != 1 {
		t.Fatalf("expected y at 2:1, got %q at %d:%d", tok.Literal, tok.Line, tok.Column)
	}
}

// TestPositions checks the line and column of tokens, including the ones
// after a newline, a comment and a multi-character token.
func TestPositions(t *testing.T) {
	input := "let x = 10;\n  y === 1; -~> note\n\tz\n\"a b\" w"

	tests := []struct {
		literal string
		line    int
		column  int
	}{
		{"let", 1, 1},
		{"x", 1, 5},
		{"=", 1, 7},
		{"10", 1, 9},
		{";", 1, 11},
		{"y", 2, 3},
		{"===", 2, 5},
		{"1", 2, 9},
		{";", 2, 10},
		{"z", 3, 2},
		{"a b", 4, 1},
		{"w", 4, 7},
		{"", 4, 8}, // EOF
		{"", 4, 8}, // EOF stays put when asked again
	}

	l := New(input)
	for i, tt := range tests {
		got := l.NextToken()
		if got.Literal != tt.literal || got.Line != tt.line || got.Column != tt.column {
			t.Fatalf("tests[%d] - expected %q at %d:%d, got %q at %d:%d",
				i, tt.literal, tt.line, tt.column, got.Literal, got.Line, got.Column)
		}
	}
}

// TestStringEscapes checks the four escapes, the unknown ones and the dot.
func TestStringEscapes(t *testing.T) {
	tests := []struct {
		input    string
		expected []token.Token
	}{
		{`"say \"hi\""`, []token.Token{{Type: token.STRING, Literal: `say "hi"`}}},
		{`"a\\b"`, []token.Token{{Type: token.STRING, Literal: `a\b`}}},
		{`"a\nb"`, []token.Token{{Type: token.STRING, Literal: "a\nb"}}},
		{`"a\tb"`, []token.Token{{Type: token.STRING, Literal: "a\tb"}}},
		{`"\\"`, []token.Token{{Type: token.STRING, Literal: `\`}}},
		{`"\\n"`, []token.Token{{Type: token.STRING, Literal: `\n`}}},
		{`"\""`, []token.Token{{Type: token.STRING, Literal: `"`}}},
		{`"añ\n🦎"`, []token.Token{{Type: token.STRING, Literal: "añ\n🦎"}}},
		{`"\q"`, []token.Token{{Type: token.ILLEGAL, Literal: `\q`}}},
		{`"ab\r cd" x`, []token.Token{{Type: token.ILLEGAL, Literal: `\r`}, {Type: token.IDENT, Literal: "x"}}},
		{`"\ñ \" x" y`, []token.Token{{Type: token.ILLEGAL, Literal: `\ñ`}, {Type: token.IDENT, Literal: "y"}}},
		{`"open\`, []token.Token{{Type: token.ILLEGAL, Literal: `"open\`}}},
		{`"open\"`, []token.Token{{Type: token.ILLEGAL, Literal: `"open\"`}}},
		{`\`, []token.Token{{Type: token.ILLEGAL, Literal: `\`}}},
		{"a.b", []token.Token{{Type: token.IDENT, Literal: "a"}, {Type: token.DOT, Literal: "."}, {Type: token.IDENT, Literal: "b"}}},
		{"a.5", []token.Token{{Type: token.IDENT, Literal: "a"}, {Type: token.DOT, Literal: "."}, {Type: token.INT, Literal: "5"}}},
		{"1.", []token.Token{{Type: token.INT, Literal: "1"}, {Type: token.DOT, Literal: "."}}},
	}

	for _, tt := range tests {
		l := New(tt.input)
		for i, want := range append(tt.expected, token.Token{Type: token.EOF, Literal: ""}) {
			got := l.NextToken()
			if got.Type != want.Type || got.Literal != want.Literal {
				t.Fatalf("input %q, token[%d] - expected=%+v, got=%+v",
					tt.input, i, want, got)
			}
		}
	}
}

// TestEscapePositions checks that the lines and columns after a string with
// escapes still count the source characters, not the decoded ones.
func TestEscapePositions(t *testing.T) {
	l := New(`"a\nb" x`)
	l.NextToken()
	if got := l.NextToken(); got.Line != 1 || got.Column != 8 {
		t.Errorf("x at line %d, column %d, want line 1, column 8", got.Line, got.Column)
	}
}

// TestFloatsAndUnits checks decimal numbers, and the units that follow a
// number. deg, rad and turn become a FLOAT in radians, px keeps the number.
func TestFloatsAndUnits(t *testing.T) {
	flt := func(s string) token.Token { return token.Token{Type: token.FLOAT, Literal: s} }
	num := func(s string) token.Token { return token.Token{Type: token.INT, Literal: s} }
	ident := func(s string) token.Token { return token.Token{Type: token.IDENT, Literal: s} }
	dot := token.Token{Type: token.DOT, Literal: "."}

	tests := []struct {
		input    string
		expected []token.Token
	}{
		{"1.5", []token.Token{flt("1.5")}},
		{"0.25", []token.Token{flt("0.25")}},
		{"10.0", []token.Token{flt("10.0")}},
		{"007.50", []token.Token{flt("007.50")}},
		{"1.5.2", []token.Token{flt("1.5"), dot, num("2")}},
		{"1.x", []token.Token{num("1"), dot, ident("x")}},
		{"1. 5", []token.Token{num("1"), dot, num("5")}},
		{".5", []token.Token{dot, num("5")}},
		{"a.5", []token.Token{ident("a"), dot, num("5")}},
		{"1.5.x", []token.Token{flt("1.5"), dot, ident("x")}},
		{"-1.5", []token.Token{{Type: token.MINUS, Literal: "-"}, flt("1.5")}},
		// Units.
		{"180deg", []token.Token{flt("3.141592653589793")}},
		{"45deg", []token.Token{flt("0.7853981633974483")}},
		{"0deg", []token.Token{flt("0.0")}},
		{"1.5rad", []token.Token{flt("1.5")}},
		{"1rad", []token.Token{flt("1.0")}},
		{"0.25turn", []token.Token{flt("1.5707963267948966")}},
		{"1turn", []token.Token{flt("6.283185307179586")}},
		{"10px", []token.Token{num("10")}},
		{"1.5px", []token.Token{flt("1.5")}},
		{"10px 20px", []token.Token{num("10"), num("20")}},
		{"10px.x", []token.Token{num("10"), dot, ident("x")}},
		// A word that is not exactly a unit is left to the identifier.
		{"5kg", []token.Token{num("5"), ident("kg")}},
		{"1.5kg x", []token.Token{flt("1.5"), ident("kg"), ident("x")}},
		{"1x", []token.Token{num("1"), ident("x")}},
		{"5degx", []token.Token{num("5"), ident("degx")}},
		{"5degrees", []token.Token{num("5"), ident("degrees")}},
		{"5Deg", []token.Token{num("5"), ident("Deg")}},
		{"5deg2", []token.Token{num("5"), ident("deg2")}},
		{"5pxy", []token.Token{num("5"), ident("pxy")}},
		{"5_", []token.Token{num("5"), ident("_")}},
		{"1e5", []token.Token{num("1"), ident("e5")}},
		// A number too big for a float becomes +Inf for the parser to reject.
		{strings.Repeat("9", 400) + "rad", []token.Token{flt("+Inf")}},
	}

	for _, tt := range tests {
		l := New(tt.input)
		for i, want := range append(tt.expected, token.Token{Type: token.EOF, Literal: ""}) {
			got := l.NextToken()
			if got.Type != want.Type || got.Literal != want.Literal {
				t.Fatalf("input %q, token[%d] - expected=%+v, got=%+v",
					tt.input, i, want, got)
			}
		}
	}
}

// TestFloatPositions checks the columns after numbers with a '.' and a unit.
func TestFloatPositions(t *testing.T) {
	l := New("1.5deg x")
	l.NextToken()
	if got := l.NextToken(); got.Line != 1 || got.Column != 8 {
		t.Errorf("x at line %d, column %d, want line 1, column 8", got.Line, got.Column)
	}
}

// TestColors checks the color literals and the lengths and digits that make
// a '#' word illegal.
func TestColors(t *testing.T) {
	color := func(s string) token.Token { return token.Token{Type: token.COLOR, Literal: s} }
	bad := func(s string) token.Token { return token.Token{Type: token.ILLEGAL, Literal: s} }
	ident := func(s string) token.Token { return token.Token{Type: token.IDENT, Literal: s} }
	comma := token.Token{Type: token.COMMA, Literal: ","}
	rparen := token.Token{Type: token.RPAREN, Literal: ")"}

	tests := []struct {
		input    string
		expected []token.Token
	}{
		{"#fff", []token.Token{color("#fff")}},
		{"#3fa34d", []token.Token{color("#3fa34d")}},
		{"#3fa34d80", []token.Token{color("#3fa34d80")}},
		{"#ABCDEF", []token.Token{color("#ABCDEF")}},
		{"#aBc", []token.Token{color("#aBc")}},
		{"#000000ff", []token.Token{color("#000000ff")}},
		{"#123,", []token.Token{color("#123"), comma}},
		{"#123)", []token.Token{color("#123"), rparen}},
		{"#fff #000", []token.Token{color("#fff"), color("#000")}},
		// The whole word is the color, so a bad one is a single token.
		{"#", []token.Token{bad("#")}},
		{"#1", []token.Token{bad("#1")}},
		{"#12", []token.Token{bad("#12")}},
		{"#1234", []token.Token{bad("#1234")}},
		{"#12345", []token.Token{bad("#12345")}},
		{"#1234567", []token.Token{bad("#1234567")}},
		{"#123456789", []token.Token{bad("#123456789")}},
		{"#ggg", []token.Token{bad("#ggg")}},
		{"#12345g", []token.Token{bad("#12345g")}},
		{"#fff_", []token.Token{bad("#fff_")}},
		{"#red", []token.Token{bad("#red")}},
		{"# fff", []token.Token{bad("#"), ident("fff")}},
	}

	for _, tt := range tests {
		l := New(tt.input)
		for i, want := range append(tt.expected, token.Token{Type: token.EOF, Literal: ""}) {
			got := l.NextToken()
			if got.Type != want.Type || got.Literal != want.Literal {
				t.Fatalf("input %q, token[%d] - expected=%+v, got=%+v",
					tt.input, i, want, got)
			}
		}
	}
}

// TestColorPositions checks the column after a color.
func TestColorPositions(t *testing.T) {
	l := New("#3fa34d x")
	l.NextToken()
	if got := l.NextToken(); got.Line != 1 || got.Column != 9 {
		t.Errorf("x at line %d, column %d, want line 1, column 9", got.Line, got.Column)
	}
}
