package fuzz

import (
	"fmt"
	"lizardscript/lexer"
	"lizardscript/parser"
	"lizardscript/token"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ParseTimeout is how long a parse may take before it counts as a hang.
// The parser fuzzer's inputs are small, so a parse takes microseconds.
const ParseTimeout = 2 * time.Second

// position is how every error starts.
var position = regexp.MustCompile(`^\[line (\d+), column (\d+)\] `)

// lines holds how many columns each line of an input has, counted the way
// the lexer counts them: one per character, not per byte, so the bytes
// after the first of a UTF-8 character don't count, but a continuation byte
// that no character started does.
type lines []int

func measure(input string) lines {
	var widths lines
	for _, line := range strings.Split(input, "\n") {
		width, continues := 0, 0
		for i := 0; i < len(line); i++ {
			b := line[i]
			if b&0xC0 == 0x80 && continues > 0 {
				continues--
				continue
			}
			width++
			switch {
			case b >= 0xF8:
				continues = 0
			case b >= 0xF0:
				continues = 3
			case b >= 0xE0:
				continues = 2
			case b >= 0xC0:
				continues = 1
			default:
				continues = 0
			}
		}
		widths = append(widths, width)
	}
	return widths
}

// inside reports whether line and column are a place in the input: a
// character of it, or the place just after the last one of a line, where
// the end of a line or of the input is.
func (l lines) inside(line, column int) bool {
	return line >= 1 && line <= len(l) && column >= 1 && column <= l[line-1]+1
}

// CheckParse parses input and lexes it on its own, and describes what is
// wrong: a panic, a parse that takes longer than ParseTimeout, an error that
// doesn't start with a position or whose position is outside the input, or
// a token whose start or end is outside the input. It gives "" when all is
// well. Errors are a normal result.
func CheckParse(input string) string {
	done := make(chan string, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- fmt.Sprintf("panic: %v", r)
			}
		}()
		done <- checkParse(input)
	}()
	select {
	case problem := <-done:
		return problem
	case <-time.After(ParseTimeout):
		return fmt.Sprintf("the parse took more than %s", ParseTimeout)
	}
}

func checkParse(input string) string {
	widths := measure(input)
	p := parser.New(lexer.New(input))
	program := p.ParseProgram()
	for _, message := range p.Errors() {
		match := position.FindStringSubmatch(message)
		if match == nil {
			return fmt.Sprintf("an error without its position: %q", message)
		}
		line, _ := strconv.Atoi(match[1])
		column, _ := strconv.Atoi(match[2])
		if !widths.inside(line, column) {
			return fmt.Sprintf("an error outside the input (%d lines): %q", len(widths), message)
		}
	}
	if len(p.Errors()) == 0 {
		// A tree without errors can be written back.
		_ = program.String()
	}

	l := lexer.New(input)
	for i := 0; i <= len(input)+1; i++ {
		tok := l.NextToken()
		if !widths.inside(tok.Line, tok.Column) || !widths.inside(tok.EndLine, tok.EndColumn) {
			return fmt.Sprintf("token %s %q at %d:%d to %d:%d is outside the input", tok.Type, tok.Literal, tok.Line, tok.Column, tok.EndLine, tok.EndColumn)
		}
		if tok.Type == token.EOF {
			return ""
		}
	}
	return "the lexer never reached the end of the input"
}
