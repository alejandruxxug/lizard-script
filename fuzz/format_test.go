package fuzz

import (
	"fmt"
	"lizardscript/format"
	"math/rand"
	"regexp"
	"strings"
	"testing"
)

// The formatter only moves the space between tokens, so formatting must
// never change what a program does. Errors move with their tokens, so
// positions are not compared; everything else is.

var (
	positions = regexp.MustCompile(`\[line \d+, column \d+\]|line \d+ column \d+|\d+:\d+$|"line": \d+, "column": \d+`)
	numbers   = regexp.MustCompile(`\d+`)
)

// behaviour is a run of input on the evaluator without its positions: the
// output, the value, the steps and the hook calls.
func behaviour(input string) string {
	program := Parse(input)
	if program == nil {
		return "doesn't parse"
	}
	o := RunEngine(program, Evaluator, true)
	var hooks []string
	for _, hook := range o.Hooks {
		// "stmt <pointer> <text>" and "layer <name> <enter> <line>:<column>".
		fields := strings.SplitN(hook, " ", 3)
		if fields[0] == "stmt" {
			hooks = append(hooks, "stmt "+fields[2])
		} else {
			hooks = append(hooks, positions.ReplaceAllString(hook, "L:C"))
		}
	}
	return fmt.Sprintf("value: %s\nsteps: %d\npanic: %s\noutput:\n%s\nhooks:\n%s",
		positions.ReplaceAllString(o.Value, "[L:C]"), o.Steps, o.Panic, positions.ReplaceAllString(o.Output, "[L:C]"), strings.Join(hooks, "\n"))
}

// CheckFormat formats input and describes what is wrong: a program that
// parses but can't be formatted, a second formatting that changes the first
// one, or a formatted program that does something else. It gives "" when
// all is well, and for input that doesn't parse.
func CheckFormat(input string) string {
	if Parse(input) == nil || KnownDifference(input) {
		return ""
	}
	out, err := format.Source(input)
	if err != nil {
		return fmt.Sprintf("a program that parses can't be formatted: %v", err)
	}
	again, err := format.Source(out)
	if err != nil {
		return fmt.Sprintf("the formatted program can't be formatted: %v\nformatted:\n%s", err, out)
	}
	if again != out {
		return fmt.Sprintf("formatting twice changes the program\nonce:\n%s\ntwice:\n%s", out, again)
	}
	before, after := behaviour(input), behaviour(out)
	if before != after && (strings.Contains(input, ".line") || strings.Contains(input, ".column")) {
		// The program prints positions, which move with the layout.
		before, after = numbers.ReplaceAllString(before, "N"), numbers.ReplaceAllString(after, "N")
	}
	if before != after {
		return fmt.Sprintf("formatting changes what the program does\nformatted:\n%s\n--- before:\n%s\n--- after:\n%s", out, before, after)
	}
	return ""
}

// FuzzFormat formats mutated programs of the repo.
func FuzzFormat(f *testing.F) {
	for _, program := range Corpus() {
		f.Add(program)
	}
	f.Fuzz(func(t *testing.T, input string) {
		if KnownBug(input) != "" {
			return
		}
		if problem := CheckFormat(input); problem != "" {
			small := ShrinkLines(input, func(s string) bool { return KnownBug(s) == "" && CheckFormat(s) != "" })
			t.Fatalf("%s\n\nminimal program:\n%s\n%s", problem, small, CheckFormat(small))
		}
	})
}

// FuzzFormatGenerated formats generated programs.
func FuzzFormatGenerated(f *testing.F) {
	random := rand.New(rand.NewSource(44))
	for i := 0; i < 40; i++ {
		data := make([]byte, 20+random.Intn(200))
		random.Read(data)
		f.Add(data)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		program := Generate(data)
		input := program.String()
		if KnownBug(input) != "" {
			return
		}
		if problem := CheckFormat(input); problem != "" {
			small := Shrink(program, func(s string) bool { return KnownBug(s) == "" && CheckFormat(s) != "" })
			t.Fatalf("%s\n\nminimal program:\n%s\n%s", problem, small, CheckFormat(small.String()))
		}
	})
}

// TestCorpusFormats formats every program of the corpus and a few hundred
// generated ones.
func TestCorpusFormats(t *testing.T) {
	for _, input := range Corpus() {
		if KnownBug(input) != "" {
			continue
		}
		if problem := CheckFormat(input); problem != "" {
			t.Errorf("%s\ninput:\n%s", problem, input)
		}
	}
	random := rand.New(rand.NewSource(2027))
	for i := 0; i < 200; i++ {
		data := make([]byte, 50+random.Intn(400))
		random.Read(data)
		input := Generate(data).String()
		if KnownBug(input) != "" {
			continue
		}
		if problem := CheckFormat(input); problem != "" {
			t.Fatalf("%s\ninput:\n%s", problem, input)
		}
	}
}
