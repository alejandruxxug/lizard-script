package lsp

import (
	"encoding/json"
	"github.com/alejandruxxug/lizard-script/lang/lexer"
	"github.com/alejandruxxug/lizard-script/terrarium"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestEveryProgramOfTheRepository asks every request about every program
// of the repository, at many places, and fails when one breaks: the server
// must answer about any code, also code with errors.
func TestEveryProgramOfTheRepository(t *testing.T) {
	var files []string
	for _, dir := range []string{"../../examples", "../../tests", "../../terrarium/std", "../../docs/flower", "testdata"} {
		filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err == nil && !d.IsDir() && strings.HasSuffix(path, ".liz") {
				files = append(files, path)
			}
			return nil
		})
	}
	if len(files) < 50 {
		t.Fatalf("only %d programs found", len(files))
	}

	s := New(Options{Globals: terrarium.CheckGlobals()})
	s.out = discard{}
	s.overlay = &overlay{texts: map[string]string{}, next: os.ReadFile}
	s.ready = true
	ask := func(method string, params any) {
		body, _ := json.Marshal(params)
		if _, err := s.call(&message{Method: method, Params: body}); err != nil && strings.Contains(err.Error(), "failed") {
			t.Errorf("%s: %v", method, err)
		}
	}
	for _, file := range files {
		text, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		abs, _ := filepath.Abs(file)
		uri := pathToURI(abs)
		document := map[string]any{"uri": uri}
		ask("textDocument/didOpen", map[string]any{"textDocument": map[string]any{"uri": uri, "version": 1, "text": string(text)}})
		for _, method := range []string{"textDocument/documentSymbol", "textDocument/foldingRange",
			"textDocument/semanticTokens/full", "textDocument/documentColor", "textDocument/formatting"} {
			ask(method, map[string]any{"textDocument": document})
		}
		d := s.docs[docKey(uri)]
		l := lexer.New(string(text))
		for i, tok := 0, l.NextToken(); tok.Type != "EOF"; i, tok = i+1, l.NextToken() {
			if i%13 != 0 {
				continue
			}
			p := map[string]any{"textDocument": document, "position": d.toPosition(tok.Line, tok.Column+1)}
			for _, method := range []string{"textDocument/hover", "textDocument/completion", "textDocument/signatureHelp",
				"textDocument/definition", "textDocument/documentHighlight"} {
				ask(method, p)
			}
		}
		ask("textDocument/didClose", map[string]any{"textDocument": document})
	}
}

type discard struct{}

func (discard) Write(p []byte) (int, error) { return len(p), nil }
