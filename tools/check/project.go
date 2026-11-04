package check

import (
	"fmt"
	"github.com/alejandruxxug/lizard-script/lang/ast"
	"github.com/alejandruxxug/lizard-script/lang/lexer"
	"github.com/alejandruxxug/lizard-script/lang/parser"
	"os"
	"path/filepath"
)

// ProjectFile is the name of the file that tells check about the files of
// a folder and the folders in it: the names their host gives them, which
// they use without defining. It is lizard-script, one hash, read and never
// run (its name keeps it apart from a module called check):
//
//	-~> The names the editor puts around these files.
//	{"names": ["canvas", "settings"]}
const ProjectFile = "lizard-check.liz"

// ProjectNames are the names of the project file nearest to the file at
// path (in its folder, or the closest folder above it that has one), and
// that project file's path; none when there is no project file.
func ProjectNames(path string) (names []string, project string, err error) {
	dir, err := filepath.Abs(filepath.Dir(path))
	if err != nil {
		return nil, "", err
	}
	for {
		project = filepath.Join(dir, ProjectFile)
		source, err := os.ReadFile(project)
		if err == nil {
			names, err := parseProject(string(source))
			if err != nil {
				return nil, project, fmt.Errorf("%s: %w", filepath.ToSlash(project), err)
			}
			return names, project, nil
		}
		if !os.IsNotExist(err) {
			return nil, project, err
		}
		up := filepath.Dir(dir)
		if up == dir {
			return nil, "", nil
		}
		dir = up
	}
}

// WithNames is globals and names, which have no members that check knows.
func WithNames(globals map[string][]string, names []string) map[string][]string {
	if len(names) == 0 {
		return globals
	}
	all := make(map[string][]string, len(globals)+len(names))
	for name, members := range globals {
		all[name] = members
	}
	for _, name := range names {
		if _, ok := all[name]; !ok {
			all[name] = nil
		}
	}
	return all
}

// parseProject reads the names of a project file: one hash literal, whose
// only key is "names", a list of strings.
func parseProject(source string) ([]string, error) {
	p := parser.New(lexer.New(source))
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		return nil, fmt.Errorf("%s", p.Errors()[0])
	}
	var hash *ast.HashLiteral
	if len(program.Statements) == 1 {
		if st, ok := program.Statements[0].(*ast.ExpressionStatement); ok {
			hash, _ = st.Expression.(*ast.HashLiteral)
		}
	}
	if hash == nil {
		return nil, fmt.Errorf(`it must be one hash, like {"names": ["canvas"]}`)
	}
	var names []string
	for _, pair := range hash.Pairs {
		key, ok := pair.Key.(*ast.StringLiteral)
		if !ok || key.Value != "names" {
			return nil, projectError(pair.Key, `the only key is "names", got %s`, pair.Key.String())
		}
		list, ok := pair.Value.(*ast.ArrayLiteral)
		if !ok {
			return nil, projectError(pair.Value, "names must be a list of strings, got %s", pair.Value.String())
		}
		for _, e := range list.Elements {
			name, ok := e.(*ast.StringLiteral)
			if !ok {
				return nil, projectError(e, "names must be a list of strings, got %s in it", e.String())
			}
			names = append(names, name.Value)
		}
	}
	return names, nil
}

// projectError is a mistake in a project file, at node.
func projectError(node ast.Node, format string, args ...any) error {
	span := node.Range()
	return fmt.Errorf("[line %d, column %d] %s", span.StartLine, span.StartColumn, fmt.Sprintf(format, args...))
}
