package fuzz

import (
	"strings"
	"testing"
)

// FuzzParser gives the parser random text, starting from the programs of
// the repo: it must never panic or hang, and every error must say where it
// is, inside the input.
func FuzzParser(f *testing.F) {
	for _, program := range Corpus() {
		f.Add(program)
	}
	for _, input := range []string{"", "(", "\"", "\"\\u{", "#", "-~>", "layer", "layer a\n  layer", "1e", "1e400", "\xff\xfe", "é\n\té", "\r\n"} {
		f.Add(input)
	}
	f.Fuzz(func(t *testing.T, input string) {
		if KnownParserBug(input) != "" {
			return
		}
		if problem := CheckParse(input); problem != "" {
			t.Fatalf("%s\ninput: %q", problem, input)
		}
	})
}

// vocabulary is what a token soup is made of: every kind of token, the
// words that are only special at the start of a statement, and some
// broken ones.
var vocabulary = strings.Fields(`let function return break continue if elif else while for true false
	try catch layer use as null x y f = + - * / // ** % == != === !== < > <= >= && || ! += -= *= /=
	, ; : . ( ) [ ] { } 0 1 42 1.5 1e3 45deg 10px 2kg #fff #12 # "s" "\n" "\u{1F98E}" "open -~>
	\ @ $ ~ ^ & | ? é`)

// soup turns each byte of data into a token and the space after it.
func soup(data []byte) string {
	var b strings.Builder
	for _, c := range data {
		b.WriteString(vocabulary[int(c)%len(vocabulary)])
		switch c >> 6 {
		case 0:
			b.WriteString(" ")
		case 1:
			b.WriteString("\n" + strings.Repeat(" ", int(c)%5))
		case 2:
			// Nothing between the two tokens.
		default:
			b.WriteString("\t")
		}
	}
	return b.String()
}

// FuzzTokenSoup gives the parser random sequences of real tokens, which get
// further into it than random text does.
func FuzzTokenSoup(f *testing.F) {
	for _, seed := range []string{"", "\x00\x01\x02", "abcdefghijklmnop", "\x20\x60\xa0\xe0\x10\x50"} {
		f.Add([]byte(seed))
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		input := soup(data)
		if KnownParserBug(input) != "" {
			return
		}
		if problem := CheckParse(input); problem != "" {
			t.Fatalf("%s\ninput: %q", problem, input)
		}
	})
}

// TestCorpusParses checks the parser fuzzers' checks against every
// program of the repo, so a plain go test runs them too.
func TestCorpusParses(t *testing.T) {
	corpus := Corpus()
	if len(corpus) < 100 {
		t.Fatalf("only %d programs in the corpus", len(corpus))
	}
	for _, input := range corpus {
		if KnownParserBug(input) != "" {
			continue
		}
		if problem := CheckParse(input); problem != "" {
			t.Errorf("%s\ninput: %q", problem, input)
		}
	}
}

func TestMeasure(t *testing.T) {
	tests := []struct {
		input        string
		line, column int
		inside       bool
	}{
		{"", 1, 1, true},
		{"", 1, 2, false},
		{"ab", 1, 3, true},
		{"ab\n", 2, 1, true},
		{"ab\n", 2, 2, false},
		{"é", 1, 2, true},
		{"é", 1, 3, false},
		{"a\nb", 0, 1, false},
		{"a\nb", 3, 1, false},
	}
	for _, tt := range tests {
		if got := measure(tt.input).inside(tt.line, tt.column); got != tt.inside {
			t.Errorf("%q: %d:%d inside is %t, expected %t", tt.input, tt.line, tt.column, got, tt.inside)
		}
	}
}
