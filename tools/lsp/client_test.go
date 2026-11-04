package lsp

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/alejandruxxug/lizard-script/terrarium"
)

// client drives a server in the same process, through the protocol, the
// way an editor does.
type client struct {
	t       *testing.T
	in      *io.PipeWriter
	out     chan *message // what the server sent, read all along so it never waits
	nextID  int
	pending []*message // notifications read while waiting for a response
	done    chan error
	texts   map[string]string // the text of each open document, by URI
}

// newClient starts a server for the folder root (the workspace) and
// initializes it.
func newClient(t *testing.T, root string, debounce time.Duration) *client {
	t.Helper()
	server := New(Options{Globals: terrarium.CheckGlobals(), Debounce: debounce})
	toServer, serverIn := io.Pipe()
	serverOut, fromServer := io.Pipe()
	c := &client{t: t, in: serverIn, out: make(chan *message, 1000), done: make(chan error, 1), texts: map[string]string{}}
	go func() {
		err := server.Serve(toServer, fromServer)
		fromServer.Close()
		c.done <- err
	}()
	go func() {
		r := bufio.NewReader(serverOut)
		for {
			m, err := readMessage(r)
			if err != nil {
				close(c.out)
				return
			}
			c.out <- m
		}
	}()
	t.Cleanup(c.close)

	var result struct {
		Capabilities map[string]any `json:"capabilities"`
	}
	c.request("initialize", map[string]any{"rootUri": pathToURI(root), "capabilities": map[string]any{}}, &result)
	if len(result.Capabilities) == 0 {
		t.Fatal("initialize gave no capabilities")
	}
	c.notify("initialized", map[string]any{})
	return c
}

// close ends the server the way an editor does.
func (c *client) close() {
	if c.in == nil {
		return
	}
	c.request("shutdown", nil, nil)
	c.notify("exit", nil)
	c.in.Close()
	c.in = nil
	select {
	case err := <-c.done:
		if err != nil {
			c.t.Errorf("Serve: %v", err)
		}
	case <-time.After(5 * time.Second):
		c.t.Error("the server didn't stop after exit")
	}
}

func (c *client) write(m *message) {
	c.t.Helper()
	if err := writeMessage(c.in, m); err != nil {
		c.t.Fatal(err)
	}
}

func (c *client) notify(method string, params any) {
	c.t.Helper()
	body, _ := json.Marshal(params)
	c.write(&message{Method: method, Params: body})
}

// call sends a request and returns its response.
func (c *client) call(method string, params any) *message {
	c.t.Helper()
	c.nextID++
	id, _ := json.Marshal(c.nextID)
	body, _ := json.Marshal(params)
	c.write(&message{ID: id, Method: method, Params: body})
	for {
		m := c.read()
		if string(m.ID) == string(id) && m.Method == "" {
			return m
		}
		c.pending = append(c.pending, m)
	}
}

// request sends a request and reads its result into result, failing the
// test on an error.
func (c *client) request(method string, params, result any) {
	c.t.Helper()
	m := c.call(method, params)
	if m.Error != nil {
		c.t.Fatalf("%s: %s", method, m.Error.Message)
	}
	if result != nil {
		if err := json.Unmarshal(m.Result, result); err != nil {
			c.t.Fatalf("%s: %v in %s", method, err, m.Result)
		}
	}
}

func (c *client) read() *message {
	c.t.Helper()
	select {
	case m, ok := <-c.out:
		if !ok {
			c.t.Fatal("the server closed its output")
		}
		return m
	case <-time.After(5 * time.Second):
		c.t.Fatal("no message from the server")
	}
	return nil
}

// diagnostics waits for the next diagnostics of uri.
func (c *client) diagnostics(uri string) []Diagnostic {
	c.t.Helper()
	for {
		var m *message
		if len(c.pending) > 0 {
			m, c.pending = c.pending[0], c.pending[1:]
		} else {
			m = c.read()
		}
		if m.Method != "textDocument/publishDiagnostics" {
			continue
		}
		var p publishDiagnosticsParams
		json.Unmarshal(m.Params, &p)
		if p.URI == uri {
			return p.Diagnostics
		}
	}
}

// open opens the file at path (relative to testdata) with its text on
// disk, or with text when it is given, and returns its URI.
func (c *client) open(path string, text ...string) string {
	c.t.Helper()
	abs, _ := filepath.Abs(filepath.Join("testdata", path))
	content := strings.Join(text, "")
	if len(text) == 0 {
		source, err := os.ReadFile(abs)
		if err != nil {
			c.t.Fatal(err)
		}
		content = string(source)
	}
	uri := pathToURI(abs)
	c.texts[uri] = content
	c.notify("textDocument/didOpen", map[string]any{"textDocument": map[string]any{
		"uri": uri, "languageId": "lizard", "version": 1, "text": content}})
	return uri
}

// change replaces the whole text of uri.
func (c *client) change(uri, text string, version int) {
	c.t.Helper()
	c.texts[uri] = text
	c.notify("textDocument/didChange", map[string]any{
		"textDocument":   map[string]any{"uri": uri, "version": version},
		"contentChanges": []map[string]any{{"text": text}}})
}

// at is the position of the "|" in marker, a piece of the text of uri with
// a "|" where the cursor goes, like "dou|ble(size)".
func (c *client) at(uri, marker string) Position {
	c.t.Helper()
	cursor := strings.Index(marker, "|")
	piece := strings.Replace(marker, "|", "", 1)
	text := c.texts[uri]
	i := strings.Index(text, piece)
	if cursor < 0 || i < 0 {
		c.t.Fatalf("%q is not in the text", marker)
	}
	offset := i + cursor
	line := strings.Count(text[:offset], "\n")
	start := strings.LastIndex(text[:offset], "\n") + 1
	return Position{Line: line, Character: utf16Len(text[start:offset])}
}

// position is the params of a request about the place of marker in uri.
func (c *client) position(uri, marker string) map[string]any {
	return map[string]any{"textDocument": map[string]any{"uri": uri}, "position": c.at(uri, marker)}
}

// textOf is the text of uri in r.
func (c *client) textOf(uri string, r Range) string {
	lines := strings.Split(c.texts[uri], "\n")
	if uri != "" && c.texts[uri] == "" {
		source, _ := os.ReadFile(uriToPath(uri))
		lines = strings.Split(string(source), "\n")
	}
	d := &document{lines: lines, text: strings.Join(lines, "\n")}
	return d.text[d.offset(r.Start):d.offset(r.End)]
}
