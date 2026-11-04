package lsp

//go:generate go run ./internal/docgen

// builtinDoc is what the documentation says about a builtin: its
// signatures, like "text.find(s, sub, [from])", and its text in Markdown.
type builtinDoc struct {
	Signatures []string
	Doc        string
}
