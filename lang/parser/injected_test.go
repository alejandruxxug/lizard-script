package parser

import (
	"github.com/alejandruxxug/lizard-script/lang/lexer"
	"github.com/alejandruxxug/lizard-script/lang/token"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"
)

// TestInjectedErrors measures how well the parser recovers. It takes every
// valid program in tests/valid and examples, breaks it with one mistake (a
// token deleted, doubled or replaced by another) at many random positions,
// and parses the result. Almost every broken program must give exactly one
// error, on a line near the mistake, and at most two. Run it with -v to see
// the rates and the programs that gave more than two.
func TestInjectedErrors(t *testing.T) {
	files := corpusFiles(t)
	random := rand.New(rand.NewSource(45))

	var broken, one, oneNear, two, more int
	for _, file := range files {
		source, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		tokens := sourceTokens(string(source))
		for range 300 {
			mistake := injectMistake(string(source), tokens, random)
			p := New(lexer.New(mistake.source))
			p.ParseProgram()
			errors := p.Errors()

			switch len(errors) {
			case 0:
				continue // the change left a valid program, like a deleted ';'
			case 1:
				one++
				if near(errors[0], mistake.line) {
					oneNear++
				}
			case 2:
				two++
			default:
				more++
				t.Logf("%s: %s gives %d errors:\n%s\n--- program:\n%s", file, mistake.what,
					len(errors), strings.Join(errors, "\n"), excerpt(mistake.source, mistake.line))
			}
			broken++
		}
	}

	if broken == 0 {
		t.Fatal("no program was broken")
	}
	rate := func(n int) float64 { return 100 * float64(n) / float64(broken) }
	t.Logf("%d files, %d broken programs: one error %.1f%% (near the mistake %.1f%%), two %.1f%%, more %.2f%%",
		len(files), broken, rate(one), rate(oneNear), rate(two), rate(more))
	if rate(oneNear) < 95 {
		t.Errorf("only %.1f%% of the broken programs give one error near the mistake, want 95%%", rate(oneNear))
	}
	if rate(more) > 0.1 {
		t.Errorf("%.2f%% of the broken programs give more than two errors, want at most 0.1%%", rate(more))
	}
}

// corpusFiles lists the programs the mistakes are injected into.
func corpusFiles(t *testing.T) []string {
	var files []string
	for _, pattern := range []string{"../../tests/valid/*.liz", "../../examples/*.liz", "../../examples/terrarium/*.liz"} {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, matches...)
	}
	if len(files) == 0 {
		t.Fatal("no programs found")
	}
	return files
}

// sourceToken is a token with its place in the source, in bytes.
type sourceToken struct {
	tok        token.Token
	start, end int
}

// sourceTokens lexes source and finds where each token is in it.
func sourceTokens(source string) []sourceToken {
	lineStarts := []int{0}
	for i := range len(source) {
		if source[i] == '\n' {
			lineStarts = append(lineStarts, i+1)
		}
	}
	// offset turns a line and a column, which counts characters, into bytes.
	offset := func(line, column int) int {
		at := lineStarts[line-1]
		for ; column > 1 && at < len(source); column-- {
			_, size := utf8.DecodeRuneInString(source[at:])
			at += size
		}
		return at
	}

	var tokens []sourceToken
	l := lexer.New(source)
	for tok := l.NextToken(); tok.Type != token.EOF; tok = l.NextToken() {
		tokens = append(tokens, sourceToken{tok, offset(tok.Line, tok.Column), offset(tok.EndLine, tok.EndColumn)})
	}
	return tokens
}

// replacements are the tokens a mistake can put in place of another one.
var replacements = []string{")", "(", "}", "{", "]", "[", ",", "=", "==", "+", ":", ";", ".", "let", "if", "x", "1", `"s"`}

// mistake is a program with one injected mistake, on line.
type mistake struct {
	source string
	line   int
	what   string
}

// injectMistake deletes, doubles or replaces one random token of source.
func injectMistake(source string, tokens []sourceToken, random *rand.Rand) mistake {
	at := tokens[random.Intn(len(tokens))]
	text := source[at.start:at.end]
	where := " at line " + strconv.Itoa(at.tok.Line) + ", column " + strconv.Itoa(at.tok.Column)

	switch random.Intn(3) {
	case 0:
		return mistake{source[:at.start] + source[at.end:], at.tok.Line, "deleting " + text + where}
	case 1:
		return mistake{source[:at.end] + " " + text + source[at.end:], at.tok.Line, "doubling " + text + where}
	default:
		other := replacements[random.Intn(len(replacements))]
		return mistake{source[:at.start] + other + source[at.end:], at.tok.Line,
			"replacing " + text + " with " + other + where}
	}
}

var errorLine = regexp.MustCompile(`^\[line (\d+), column \d+\]`)

// near reports whether message is on a line at most two lines away from line.
func near(message string, line int) bool {
	match := errorLine.FindStringSubmatch(message)
	if match == nil {
		return false
	}
	got, _ := strconv.Atoi(match[1])
	return got >= line-2 && got <= line+2
}

// excerpt shows the lines of source around line, for a failure message.
func excerpt(source string, line int) string {
	lines := strings.Split(source, "\n")
	from, to := max(line-4, 0), min(line+3, len(lines))
	return strings.Join(lines[from:to], "\n")
}
