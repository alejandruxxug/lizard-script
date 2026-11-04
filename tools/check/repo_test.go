package check_test

import (
	"flag"
	"fmt"
	"github.com/alejandruxxug/lizard-script/internal/corpus"
	"github.com/alejandruxxug/lizard-script/lang/ast"
	"github.com/alejandruxxug/lizard-script/lang/lexer"
	"github.com/alejandruxxug/lizard-script/lang/parser"
	"github.com/alejandruxxug/lizard-script/terrarium"
	_ "github.com/alejandruxxug/lizard-script/terrarium/window" // the guide's programs use window and pen
	"github.com/alejandruxxug/lizard-script/tools/check"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "rewrite testdata/intended.txt with what check finds now")

// The programs of the repository have no findings but the intended ones,
// listed in testdata/intended.txt: a test of an error that a try catches,
// a guide's example of a mistake, a comparison of literals that shows what
// == does. Each line is "program: severity rule: message", without the
// position, so editing a program elsewhere doesn't change the list.
//
// The guide's code blocks are checked as its reader runs them: a block
// knows the names the blocks before it in its chapter define. Blocks with
// syntax errors (the guide shows those too) are left out, and so are the
// programs of examples/errors, which are broken on purpose.
func TestRepositoryHasOnlyIntendedFindings(t *testing.T) {
	var found []string
	for _, p := range corpus.Files("tests/valid", "examples", "terrarium/lib") {
		if strings.HasPrefix(p.Name, "examples/errors/") {
			continue
		}
		found = append(found, findings(p.Name, filepath.Join(corpus.Root(), p.Name), p.Source, terrarium.CheckGlobals())...)
	}
	chapter, globals := "", map[string][]string{}
	for _, p := range corpus.GuideBlocks() {
		file := p.Name[:strings.LastIndex(p.Name, ":")]
		if file != chapter {
			chapter, globals = file, copyGlobals(terrarium.CheckGlobals())
		}
		program, ok := parse(p.Source)
		if !ok {
			continue
		}
		found = append(found, findings(file, filepath.Join(corpus.Root(), file), p.Source, globals)...)
		for _, name := range topNames(program) {
			globals[name] = nil
		}
	}
	sort.Strings(found)

	listFile := filepath.Join("testdata", "intended.txt")
	if *update {
		header := "# What check finds in the repository's programs, all of it on purpose (see repo_test.go).\n"
		if err := os.WriteFile(listFile, []byte(header+strings.Join(found, "\n")+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	text, err := os.ReadFile(listFile)
	if err != nil {
		t.Fatal(err)
	}
	var intended []string
	for _, line := range strings.Split(strings.TrimSpace(string(text)), "\n") {
		if line != "" && !strings.HasPrefix(line, "#") {
			intended = append(intended, line)
		}
	}
	for _, line := range difference(found, intended) {
		t.Errorf("not intended: %s", line)
	}
	for _, line := range difference(intended, found) {
		t.Errorf("intended but not found any more: %s", line)
	}
}

// findings checks source, the program called name at path, and writes each
// finding as "name: severity rule: message".
func findings(name, path, source string, globals map[string][]string) []string {
	var lines []string
	for _, d := range check.SourceWith(filepath.ToSlash(path), source, globals) {
		message := strings.ReplaceAll(d.Message, filepath.ToSlash(corpus.Root())+"/", "")
		lines = append(lines, fmt.Sprintf("%s: %s %s: %s", name, d.Severity, d.Rule, message))
	}
	return lines
}

func parse(source string) (*ast.Program, bool) {
	p := parser.New(lexer.New(source))
	program := p.ParseProgram()
	return program, len(p.Errors()) == 0
}

// topNames are the names program defines at its top level.
func topNames(program *ast.Program) []string {
	var names []string
	for _, st := range program.Statements {
		if let, ok := st.(*ast.LetStatement); ok {
			names = append(names, let.Name.Value)
		}
	}
	return names
}

func copyGlobals(globals map[string][]string) map[string][]string {
	out := map[string][]string{}
	for name, members := range globals {
		out[name] = members
	}
	return out
}

// difference is the lines of a that b doesn't have, counting repeats.
func difference(a, b []string) []string {
	count := map[string]int{}
	for _, line := range b {
		count[line]++
	}
	var out []string
	for _, line := range a {
		if count[line] > 0 {
			count[line]--
		} else {
			out = append(out, line)
		}
	}
	return out
}
