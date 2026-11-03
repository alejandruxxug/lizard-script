// Package lsp is the language server of lizard-script: lizardscript lsp
// speaks the Language Server Protocol over the standard input and output,
// so an editor (VS Code, Zed, Neovim, Helix, ...) can show the mistakes
// package check finds while the code is written, format it with package
// format, and complete, explain and find its names.
//
// The server reads the code; it never runs it. It knows the names of a file
// the way the evaluator finds them (see index.go), the modules the file
// uses, and the builtins and the Terrarium from their documentation
// (builtin_docs.go, made from docs/SEMANTICS.md and docs/TERRARIUM.md).
package lsp

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"lizardscript/check"
	"lizardscript/evaluator"
	"lizardscript/object"
	"path/filepath"
	"sync"
	"time"
)

// Options are the choices of the host that starts a server.
type Options struct {
	// Globals are the names the host gives a program, with the members of
	// each module, besides the builtins: terrarium.CheckGlobals() for
	// lizardscript.
	Globals map[string][]string
	// Debounce is how long the diagnostics of a file wait after a change,
	// so they are not made again for each key. 0 makes them at once.
	Debounce time.Duration
}

// Server is one language server, for one editor.
type Server struct {
	opts    Options
	globals map[string][]string // Options.Globals and the builtins

	mu        sync.Mutex
	docs      map[string]*document // by path, or by URI when it is not a file
	loader    *loader
	overlay   *overlay
	roots     []string // the folders of the workspace
	workspace map[string]*workspaceFile
	timers    map[string]*time.Timer
	ready     bool // initialize was answered
	stopping  bool // shutdown was asked

	outMu sync.Mutex
	out   io.Writer
}

// handler answers one request or notification.
type handler func(s *Server, params json.RawMessage) (any, error)

var handlers map[string]handler

func init() {
	handlers = map[string]handler{
		"initialize":                          (*Server).initialize,
		"initialized":                         nothing,
		"shutdown":                            (*Server).shutdown,
		"textDocument/didOpen":                (*Server).didOpen,
		"textDocument/didChange":              (*Server).didChange,
		"textDocument/didClose":               (*Server).didClose,
		"textDocument/didSave":                nothing,
		"textDocument/formatting":             locked((*Server).formatting),
		"textDocument/completion":             locked((*Server).completion),
		"textDocument/hover":                  locked((*Server).hover),
		"textDocument/signatureHelp":          locked((*Server).signatureHelp),
		"textDocument/definition":             locked((*Server).definition),
		"textDocument/references":             locked((*Server).references),
		"textDocument/documentHighlight":      locked((*Server).highlights),
		"textDocument/prepareRename":          locked((*Server).prepareRename),
		"textDocument/rename":                 locked((*Server).rename),
		"textDocument/documentSymbol":         locked((*Server).documentSymbols),
		"workspace/symbol":                    locked((*Server).workspaceSymbols),
		"textDocument/foldingRange":           locked((*Server).foldingRanges),
		"textDocument/semanticTokens/full":    locked((*Server).semanticTokens),
		"textDocument/documentColor":          locked((*Server).documentColors),
		"textDocument/colorPresentation":      locked((*Server).colorPresentations),
		"workspace/didChangeConfiguration":    nothing,
		"workspace/didChangeWatchedFiles":     (*Server).filesChanged,
		"workspace/didChangeWorkspaceFolders": nothing,
		"$/cancelRequest":                     nothing,
		"$/setTrace":                          nothing,
	}
}

func nothing(*Server, json.RawMessage) (any, error) { return nil, nil }

// locked runs h with the server's state to itself. The handlers that
// change documents lock it themselves, since they publish after.
func locked(h handler) handler {
	return func(s *Server, params json.RawMessage) (any, error) {
		s.mu.Lock()
		defer s.mu.Unlock()
		return h(s, params)
	}
}

// New makes a server.
func New(opts Options) *Server {
	globals := builtinGlobals()
	for name, members := range opts.Globals {
		globals[name] = members
	}
	return &Server{opts: opts, globals: globals, docs: map[string]*document{},
		loader: newLoader(globals), workspace: map[string]*workspaceFile{}, timers: map[string]*time.Timer{}}
}

// Serve answers the messages of in on out until the editor says exit, or
// in ends. While it runs, a use statement reads the files the editor has
// open from the editor, not from the disk.
func (s *Server) Serve(in io.Reader, out io.Writer) error {
	s.out = out
	s.overlay = &overlay{texts: map[string]string{}, next: evaluator.ReadModule}
	evaluator.ReadModule = s.overlay.read
	defer func() { evaluator.ReadModule = s.overlay.next }()

	r := bufio.NewReader(in)
	for {
		m, err := readMessage(r)
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		if m.Method == "exit" {
			s.stopTimers()
			return nil
		}
		s.handle(m)
	}
}

// handle answers one message: a request gets a response, a notification
// none. A response from the editor (to a request of the server) is
// ignored: the server asks nothing.
func (s *Server) handle(m *message) {
	if m.Error != nil {
		s.send(&message{ID: json.RawMessage("null"), Error: m.Error})
		return
	}
	if m.Method == "" {
		return
	}
	isRequest := len(m.ID) > 0
	result, err := s.call(m)
	if !isRequest {
		return
	}
	reply := &message{ID: m.ID}
	var rerr *responseError
	switch {
	case errors.As(err, &rerr):
		reply.Error = rerr
	case err != nil:
		reply.Error = &responseError{codeRequestFailed, err.Error()}
	default:
		reply.Result, _ = json.Marshal(result)
	}
	s.send(reply)
}

// call runs the handler of m's method.
func (s *Server) call(m *message) (result any, err error) {
	h, ok := handlers[m.Method]
	if !ok {
		return nil, &responseError{codeMethodNotFound, "unknown method " + m.Method}
	}
	s.mu.Lock()
	ready := s.ready || m.Method == "initialize"
	s.mu.Unlock()
	if !ready {
		return nil, &responseError{codeNotInitialized, "the server was not initialized"}
	}
	defer func() {
		// A bug of the server answers one request with an error instead of
		// closing the editor's connection.
		if r := recover(); r != nil {
			result, err = nil, fmt.Errorf("lsp: %s failed: %v", m.Method, r)
		}
	}()
	return h(s, m.Params)
}

// send writes one message to the editor.
func (s *Server) send(m *message) {
	s.outMu.Lock()
	defer s.outMu.Unlock()
	writeMessage(s.out, m)
}

// notify sends a notification.
func (s *Server) notify(method string, params any) {
	body, _ := json.Marshal(params)
	s.send(&message{Method: method, Params: body})
}

// decode reads the params of a request into v.
func decode(params json.RawMessage, v any) error {
	if err := json.Unmarshal(params, v); err != nil {
		return &responseError{codeInvalidParams, err.Error()}
	}
	return nil
}

// The semantic token types and modifiers the server gives, in the order of
// its legend. color and point are lizard's own: an editor colors them with
// the scopes its extension maps them to.
var (
	tokenTypes     = []string{"keyword", "function", "variable", "parameter", "property", "namespace", "number", "string", "comment", "operator", "color", "point", "label"}
	tokenModifiers = []string{"declaration", "defaultLibrary", "readonly"}
)

func (s *Server) initialize(params json.RawMessage) (any, error) {
	var p initializeParams
	if err := decode(params, &p); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, folder := range p.WorkspaceFolders {
		if path := uriToPath(folder.URI); path != "" {
			s.roots = append(s.roots, path)
		}
	}
	if len(s.roots) == 0 {
		if path := uriToPath(p.RootURI); path != "" {
			s.roots = append(s.roots, path)
		}
	}
	s.ready = true
	return map[string]any{
		"capabilities": map[string]any{
			"textDocumentSync":           map[string]any{"openClose": true, "change": 2, "save": true},
			"documentFormattingProvider": true,
			"completionProvider":         map[string]any{"triggerCharacters": []string{"."}},
			"hoverProvider":              true,
			"signatureHelpProvider":      map[string]any{"triggerCharacters": []string{"(", ","}},
			"definitionProvider":         true,
			"referencesProvider":         true,
			"documentHighlightProvider":  true,
			"renameProvider":             map[string]any{"prepareProvider": true},
			"documentSymbolProvider":     true,
			"workspaceSymbolProvider":    true,
			"foldingRangeProvider":       true,
			"colorProvider":              true,
			"semanticTokensProvider": map[string]any{
				"legend": map[string]any{"tokenTypes": tokenTypes, "tokenModifiers": tokenModifiers},
				"full":   true,
			},
		},
		"serverInfo": map[string]any{"name": "lizardscript", "version": "0.1.0"},
	}, nil
}

func (s *Server) shutdown(json.RawMessage) (any, error) {
	s.mu.Lock()
	s.stopping = true
	s.mu.Unlock()
	s.stopTimers()
	return nil, nil
}

func (s *Server) stopTimers() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for key, t := range s.timers {
		t.Stop()
		delete(s.timers, key)
	}
}

// docKey is how the server knows a document: its file, or its URI.
func docKey(uri string) string {
	if path := uriToPath(uri); path != "" {
		return path
	}
	return uri
}

func (s *Server) didOpen(params json.RawMessage) (any, error) {
	var p didOpenParams
	if err := decode(params, &p); err != nil {
		return nil, err
	}
	key := docKey(p.TextDocument.URI)
	s.mu.Lock()
	s.docs[key] = newDocument(p.TextDocument.URI, uriToPath(p.TextDocument.URI), p.TextDocument.Text, p.TextDocument.Version)
	s.changed(key)
	s.mu.Unlock()
	s.publish(key)
	return nil, nil
}

func (s *Server) didChange(params json.RawMessage) (any, error) {
	var p didChangeParams
	if err := decode(params, &p); err != nil {
		return nil, err
	}
	key := docKey(p.TextDocument.URI)
	s.mu.Lock()
	d := s.docs[key]
	if d == nil {
		s.mu.Unlock()
		return nil, nil
	}
	for _, change := range p.ContentChanges {
		d.edit(change.Range, change.Text)
	}
	d.version = p.TextDocument.Version
	s.changed(key)
	s.mu.Unlock()
	s.schedule(key)
	return nil, nil
}

func (s *Server) didClose(params json.RawMessage) (any, error) {
	var p didCloseParams
	if err := decode(params, &p); err != nil {
		return nil, err
	}
	key := docKey(p.TextDocument.URI)
	s.mu.Lock()
	delete(s.docs, key)
	if t := s.timers[key]; t != nil {
		t.Stop()
		delete(s.timers, key)
	}
	s.overlay.remove(key)
	s.forget()
	s.mu.Unlock()
	// A closed file keeps no mistakes in the editor's list.
	s.notify("textDocument/publishDiagnostics", publishDiagnosticsParams{URI: p.TextDocument.URI, Diagnostics: []Diagnostic{}})
	return nil, nil
}

// filesChanged forgets what the server read from the disk.
func (s *Server) filesChanged(json.RawMessage) (any, error) {
	s.mu.Lock()
	s.workspace = map[string]*workspaceFile{}
	s.forget()
	s.mu.Unlock()
	return nil, nil
}

// changed notes that the text of the document key changed: a file that
// uses it as a module must be read again too.
func (s *Server) changed(key string) {
	d := s.docs[key]
	if d.path != "" {
		s.overlay.set(d.path, d.text)
	}
	s.forget()
}

// forget drops every analysis, which are made again when needed.
func (s *Server) forget() {
	for _, d := range s.docs {
		d.index = nil
	}
}

// schedule makes the diagnostics of key after the debounce time.
func (s *Server) schedule(key string) {
	if s.opts.Debounce <= 0 {
		s.publish(key)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if t := s.timers[key]; t != nil {
		t.Stop()
	}
	s.timers[key] = time.AfterFunc(s.opts.Debounce, func() { s.publish(key) })
}

// publish sends the diagnostics of the document key.
func (s *Server) publish(key string) {
	s.mu.Lock()
	d := s.docs[key]
	if d == nil || s.stopping {
		s.mu.Unlock()
		return
	}
	params := publishDiagnosticsParams{URI: d.uri, Diagnostics: s.diagnostics(d)}
	version := d.version
	params.Version = &version
	s.mu.Unlock()
	s.notify("textDocument/publishDiagnostics", params)
}

// diagnostics are what package check finds in d, with the names of its
// project file (check.ProjectNames; one with mistakes is left out here,
// lizardscript check reports them).
func (s *Server) diagnostics(d *document) []Diagnostic {
	out := []Diagnostic{}
	globals := s.opts.Globals
	if d.path != "" {
		if names, _, err := check.ProjectNames(d.path); err == nil {
			globals = check.WithNames(globals, names)
		}
	}
	for _, found := range check.SourceWith(d.path, d.text, globals) {
		severity := 1
		if found.Severity == check.Warning {
			severity = 2
		}
		message := found.Message
		if found.Hint != "" {
			message += "\nhint: " + found.Hint
		}
		diagnostic := Diagnostic{Severity: severity, Code: found.Rule, Source: "lizard", Message: message}
		if found.Line > 0 {
			diagnostic.Range = d.toRange(found.Line, found.Column, found.EndLine, found.EndColumn)
		}
		switch found.Rule {
		case check.RuleUnused, check.RuleUnreachable, check.RuleUnusedModule:
			diagnostic.Tags = []int{1} // shown faded
		}
		out = append(out, diagnostic)
	}
	return out
}

// document is the open document of uri, or an error.
func (s *Server) document(uri string) (*document, error) {
	d := s.docs[docKey(uri)]
	if d == nil {
		return nil, &responseError{codeInvalidParams, "the document is not open: " + uri}
	}
	return d, nil
}

// indexOf is the analysis of d, made when first needed.
func (s *Server) indexOf(d *document) *index {
	if d.index == nil {
		name := d.path
		if name == "" {
			name = d.uri
		}
		d.index = newIndex(name, d.text, s.globals, s.loader)
	}
	return d.index
}

// at is the document of a request, its analysis and the position in the
// language's terms.
func (s *Server) at(p TextDocumentPositionParams) (*document, *index, pos, error) {
	d, err := s.document(p.TextDocument.URI)
	if err != nil {
		return nil, nil, pos{}, err
	}
	line, column := d.fromPosition(p.Position)
	return d, s.indexOf(d), pos{line, column}, nil
}

// builtinGlobals are the names every program has: the builtins, null and
// the libraries, with the members of the libraries.
func builtinGlobals() map[string][]string {
	globals := map[string][]string{}
	for _, name := range evaluator.BuiltinNames() {
		value, _ := evaluator.Builtin(name)
		var members []string
		if h, ok := value.(*object.Hash); ok {
			h.Each(func(pair object.HashPair) {
				if key, ok := pair.Key.(*object.String); ok {
					members = append(members, key.Value)
				}
			})
		}
		globals[name] = members
	}
	return globals
}

// sameFile reports whether two source names are the same file.
func sameFile(a, b string) bool {
	return filepath.Clean(filepath.FromSlash(a)) == filepath.Clean(filepath.FromSlash(b))
}
