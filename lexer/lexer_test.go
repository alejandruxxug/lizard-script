package lexer

import (
	"lizardscript/token"
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
			if got != want {
				t.Fatalf("input %q, token[%d] - expected=%+v, got=%+v",
					tt.input, i, want, got)
			}
		}
	}
}
