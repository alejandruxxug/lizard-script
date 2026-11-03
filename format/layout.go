package format

import (
	"lizardscript/lexer"
	"lizardscript/token"
	"strings"
	"unicode/utf8"
)

// item is a token with the text it was written with (a string with its
// quotes and escapes, 45deg as 45deg), which is what the layout prints.
type item struct {
	token.Token
	text string
}

// readTokens is every token of text, comments included, without the EOF.
func readTokens(text string) []item {
	lines := strings.Split(text, "\n")
	var items []item
	l := lexer.New(text)
	l.KeepComments()
	for tok := l.NextToken(); tok.Type != token.EOF; tok = l.NextToken() {
		written := between(lines, tok.Line, tok.Column, tok.EndLine, tok.EndColumn)
		if tok.Type == token.COMMENT {
			written = strings.TrimRight(written, " \t\r")
		}
		items = append(items, item{tok, written})
	}
	return items
}

// between is the text from (line, column) up to (endLine, endColumn).
func between(lines []string, line, column, endLine, endColumn int) string {
	var parts []string
	for l := line; l <= endLine && l <= len(lines); l++ {
		text := lines[l-1]
		from, to := 0, len(text)
		if l == line {
			from = offset(text, column)
		}
		if l == endLine {
			to = offset(text, endColumn)
		}
		parts = append(parts, text[from:max(from, to)])
	}
	return strings.Join(parts, "\n")
}

// offset is the byte where column starts in text, counted like the lexer
// counts columns: one for each byte that starts a character. Bytes that are
// not valid UTF-8 are kept as they are.
func offset(text string, column int) int {
	for i := 0; i < len(text); i++ {
		if text[i]&0xC0 != 0x80 {
			column--
			if column == 0 {
				return i
			}
		}
	}
	return len(text)
}

// line is one line of the result.
type line struct {
	indent  int
	code    strings.Builder
	comment string // the comment at its end, if any
	blank   bool
}

// open is a bracket, block or layer body that is open at some point of the
// layout, with the indentation of the line it was opened on.
type open struct {
	kind   byte // '(' '[', 'b' for a block, 'h' for a hash, 'l' for a layer body
	indent int
	layer  layerBody
}

// printer lays out the tokens, one at a time.
type printer struct {
	items   []item
	facts   *facts
	kinds   map[int]byte // for each brace, by its index: 'b' or 'h'
	openers map[int]int  // for each '}', the index of its '{'
	lines   []*line
	stack   []open
	prev    int // index of the last token printed, -1 before the first
	layer   *layerBody
}

// layout returns the formatted code of items, whose tree gave f.
func layout(items []item, f *facts) string {
	kinds, openers := braces(items, f)
	pr := &printer{items: items, facts: f, kinds: kinds, openers: openers, prev: -1}
	for i := range items {
		pr.print(i)
	}
	alignComments(pr.lines)
	return render(pr.lines)
}

// braces tells, for each '{' and '}', whether it is a block's or a hash's,
// and for each '}' which '{' it closes. A '}' is of the kind of its '{'.
func braces(items []item, f *facts) (map[int]byte, map[int]int) {
	kinds, openers := map[int]byte{}, map[int]int{}
	var opened []int
	for i, it := range items {
		switch it.Type {
		case token.LBRACE:
			kind := byte('b')
			if f.hashes[tokenPos(it.Token)] {
				kind = 'h'
			}
			kinds[i] = kind
			opened = append(opened, i)
		case token.RBRACE:
			if n := len(opened); n > 0 {
				kinds[i], openers[i] = kinds[opened[n-1]], opened[n-1]
				opened = opened[:n-1]
			}
		}
	}
	return kinds, openers
}

// print adds token i to the layout: on a new line or after the one before.
func (pr *printer) print(i int) {
	it := pr.items[i]
	pr.closeLayers(it)

	switch {
	case pr.startsLine(i):
		pr.newLine(i)
		pr.current().code.WriteString(it.text)
	case it.Type == token.COMMENT:
		pr.current().comment = it.text // spaced by alignComments
	default:
		if pr.spaceBefore(i) {
			pr.current().code.WriteByte(' ')
		}
		pr.current().code.WriteString(it.text)
	}
	pr.track(i)
	pr.prev = i
}

func (pr *printer) current() *line { return pr.lines[len(pr.lines)-1] }

// startsLine reports whether token i goes at the start of a line: where it
// was, unless it joins the line before, and where a rule breaks the line.
func (pr *printer) startsLine(i int) bool {
	if pr.prev < 0 {
		return true
	}
	it := pr.items[i]
	if pr.items[pr.prev].Type == token.COMMENT {
		return true
	}
	if it.LineStart {
		return !pr.joins(i)
	}
	return pr.facts.breaks[tokenPos(it.Token)]
}

// joins reports whether token i, which starts its line, goes on the line
// before: a block's '{', an else or elif after the '}' of a block over
// several lines, the catch of such a try, the '}' of an empty hash. The parser reads all of them across a line
// break, so joining them doesn't change the program.
func (pr *printer) joins(i int) bool {
	it, prev := pr.items[i], pr.items[pr.prev]
	switch {
	case it.Type == token.LBRACE:
		return pr.kinds[i] == 'b'
	case it.Type == token.ELSE || it.Type == token.ELIF:
		return prev.Type == token.RBRACE && pr.spansLines(pr.prev)
	case it.Type == token.IDENT && it.Literal == "catch":
		return pr.facts.tryEnds[tokenPos(prev.Token)] && pr.spansLines(pr.prev)
	}
	return pr.facts.joins[tokenPos(it.Token)]
}

// spansLines reports whether the block that the '}' at index i closes was
// written over several lines. One written on one line keeps the else or
// catch after it on the next line, so a chain of short blocks stays one per
// line.
func (pr *printer) spansLines(i int) bool {
	return pr.items[pr.openers[i]].Line != pr.items[i].Line
}

// newLine starts the line of token i, after a blank line when the source
// had one or more there.
func (pr *printer) newLine(i int) {
	it := pr.items[i]
	if pr.prev >= 0 && it.Line-pr.items[pr.prev].EndLine > 1 && it.LineStart {
		pr.lines = append(pr.lines, &line{blank: true})
	}
	pr.lines = append(pr.lines, &line{indent: pr.indent(i)})
}

// indent is the indentation of the line that token i starts. Inside a
// bracket it is one more than the line that opened it. Inside a block, a
// layer body or the program a statement gets that too, and the next lines
// of a statement that goes on one more. A closing bracket goes back to the
// line of its opening one.
func (pr *printer) indent(i int) int {
	it := pr.items[i]
	var top *open
	if n := len(pr.stack); n > 0 {
		top = &pr.stack[n-1]
	}
	if top == nil {
		return pr.statementIndent(i, 0)
	}
	switch it.Type {
	case token.RPAREN, token.RBRACKET, token.RBRACE:
		return top.indent
	}
	switch top.kind {
	case 'b', 'l':
		return pr.statementIndent(i, top.indent+1)
	case 'h':
		return pr.pairIndent(it, top.indent+1)
	}
	return top.indent + 1
}

// pairIndent is the indentation of token it in a hash over several lines,
// whose pairs each start a line indented by base.
func (pr *printer) pairIndent(it item, base int) int {
	if pr.facts.breaks[tokenPos(it.Token)] || it.Type == token.COMMENT {
		return base
	}
	return base + 1
}

// statementIndent is the indentation of token i among statements whose
// own indentation is base: base for a statement, a comment, and the parts
// of an if or a try that start a line (a comment kept them there), and one
// more for the next lines of a statement.
func (pr *printer) statementIndent(i, base int) int {
	it := pr.items[i]
	switch {
	case pr.facts.statements[tokenPos(it.Token)], it.Type == token.COMMENT, it.Type == token.SEMICOLON,
		it.Type == token.ELSE, it.Type == token.ELIF, it.Literal == "catch", pr.kinds[i] == 'b':
		return base
	}
	return base + 1
}

// spaceBefore reports whether a space goes between token i and the one
// printed before it, on the same line.
func (pr *printer) spaceBefore(i int) bool {
	it, prev := pr.items[i], pr.items[pr.prev]
	switch prev.Type {
	case token.LPAREN, token.LBRACKET, token.DOT:
		return false
	}
	switch it.Type {
	case token.RPAREN, token.RBRACKET, token.COMMA, token.SEMICOLON, token.COLON, token.DOT:
		return false
	case token.LPAREN:
		if pr.facts.calls[tokenPos(it.Token)] || pr.afterFunction() {
			return false
		}
	case token.LBRACKET:
		if pr.facts.indexes[tokenPos(it.Token)] {
			return false
		}
	case token.RBRACE:
		return pr.kinds[i] == 'b' && prev.Type != token.LBRACE
	}
	if prev.Type == token.LBRACE {
		return pr.kinds[pr.prev] == 'b'
	}
	return !pr.facts.prefix[tokenPos(prev.Token)]
}

// afterFunction reports whether the token printed last is the word
// function, or the name after it: the '(' that follows opens the parameters.
func (pr *printer) afterFunction() bool {
	prev := pr.items[pr.prev]
	if prev.Type == token.FUNCTION {
		return true
	}
	return prev.Type == token.IDENT && pr.prev > 0 && pr.items[pr.prev-1].Type == token.FUNCTION
}

// track keeps the stack of open brackets, blocks and layer bodies up to
// date after token i.
func (pr *printer) track(i int) {
	it := pr.items[i]
	indent := pr.current().indent
	switch it.Type {
	case token.LPAREN, token.LBRACKET:
		pr.stack = append(pr.stack, open{kind: it.Type[0], indent: indent})
	case token.LBRACE:
		pr.stack = append(pr.stack, open{kind: pr.kinds[i], indent: indent})
	case token.RPAREN, token.RBRACKET, token.RBRACE:
		if n := len(pr.stack); n > 0 {
			pr.stack = pr.stack[:n-1]
		}
	}

	// A layer's body starts after its name.
	if pr.layer != nil {
		pr.stack = append(pr.stack, open{kind: 'l', indent: indent, layer: *pr.layer})
		pr.layer = nil
	} else if body, ok := pr.facts.layers[tokenPos(it.Token)]; ok {
		pr.layer = &body
	}
}

// closeLayers ends the layer bodies that end before it. A comment that
// starts a line further right than the word layer still belongs to the
// body, as it was written.
func (pr *printer) closeLayers(it item) {
	for n := len(pr.stack); n > 0 && pr.stack[n-1].kind == 'l'; n = len(pr.stack) {
		body := pr.stack[n-1].layer
		if tokenPos(it.Token).before(body.end) {
			return
		}
		if it.Type == token.COMMENT && it.LineStart && it.Column > body.column {
			return
		}
		pr.stack = pr.stack[:n-1]
	}
}

// alignComments starts the end-of-line comments of consecutive lines in
// one column, one space after the longest code among them. A line with a
// string over several lines ends the group.
func alignComments(lines []*line) {
	for start := 0; start < len(lines); {
		end := start
		for end < len(lines) && aligns(lines[end]) {
			end++
		}
		if end == start {
			start++
			continue
		}
		width := 0
		for _, l := range lines[start:end] {
			width = max(width, codeWidth(l))
		}
		for _, l := range lines[start:end] {
			l.comment = strings.Repeat(" ", width-codeWidth(l)+1) + l.comment
		}
		start = end
	}
}

func aligns(l *line) bool {
	return l.comment != "" && !strings.Contains(l.code.String(), "\n")
}

func codeWidth(l *line) int {
	return 2*l.indent + utf8.RuneCountInString(l.code.String())
}

// render writes the lines, without blank lines at the start or the end,
// and with one line break at the end.
func render(lines []*line) string {
	for len(lines) > 0 && lines[0].blank {
		lines = lines[1:]
	}
	for len(lines) > 0 && lines[len(lines)-1].blank {
		lines = lines[:len(lines)-1]
	}
	var out strings.Builder
	for _, l := range lines {
		if !l.blank {
			out.WriteString(strings.Repeat("  ", l.indent))
			out.WriteString(l.code.String())
			out.WriteString(l.comment)
		}
		out.WriteByte('\n')
	}
	return out.String()
}
