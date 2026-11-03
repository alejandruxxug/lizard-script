package object

import "testing"

func TestQuote(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"", `""`},
		{"abc", `"abc"`},
		{`say "hi"`, `"say \"hi\""`},
		{`a\b`, `"a\\b"`},
		{"two\nlines", `"two\nlines"`},
		{"tab\there\r", `"tab\there\r"`},
		{"\x01\x7f", `"\u{1}\u{7f}"`},
		{"ñandú 🦎", `"ñandú 🦎"`},
	}
	for _, tt := range tests {
		if got := Quote(tt.input); got != tt.expected {
			t.Errorf("Quote(%q): expected %s, got %s", tt.input, tt.expected, got)
		}
	}
}

func TestInspectQuotesNestedStrings(t *testing.T) {
	str := func(s string) Object { return &String{Value: s} }
	inner := &Array{Elements: []Object{str(`q"`)}}
	hash := NewHash()
	hash.Set(&String{Value: "k"}, str("a\nb"))

	tests := []struct {
		value    Object
		expected string
	}{
		{str(`plain "text"`), `plain "text"`},
		{&Array{Elements: []Object{&Integer{Value: 1}, str("a")}}, `[1, "a"]`},
		{&Array{Elements: []Object{inner}}, `[["q\""]]`},
		{hash, `{"k": "a\nb"}`},
	}
	for _, tt := range tests {
		if got := tt.value.Inspect(); got != tt.expected {
			t.Errorf("expected %s, got %s", tt.expected, got)
		}
	}
}

func TestHashVersion(t *testing.T) {
	key := func(s string) *String { return &String{Value: s} }
	hash := NewHash()
	tests := []struct {
		name    string
		change  func()
		changes bool
	}{
		{"set a new key", func() { hash.Set(key("a"), &Integer{Value: 1}) }, true},
		{"set an existing key", func() { hash.Set(key("a"), &Integer{Value: 2}) }, true},
		{"get", func() { hash.Get(key("a")) }, false},
		{"copy", func() { hash.Copy().Set(key("b"), &Integer{Value: 3}) }, false},
		{"delete a missing key", func() { hash.Delete(key("b")) }, false},
		{"delete a key", func() { hash.Delete(key("a")) }, true},
	}
	for _, tt := range tests {
		before := hash.Version()
		tt.change()
		if changed := hash.Version() != before; changed != tt.changes {
			t.Errorf("%s: version changed is %t, expected %t", tt.name, changed, tt.changes)
		}
	}
}

func TestHashCopy(t *testing.T) {
	key := func(s string) *String { return &String{Value: s} }
	shared := &Array{}
	hash := NewHash()
	hash.Set(key("b"), &Integer{Value: 1})
	hash.Set(key("a"), shared)

	copied := hash.Copy()
	copied.Set(key("c"), &Integer{Value: 3})
	copied.Delete(key("b"))

	tests := []struct {
		value    *Hash
		expected string
		length   int
	}{
		{hash, `{"b": 1, "a": []}`, 2},
		{copied, `{"a": [], "c": 3}`, 2},
		{(&Hash{}).Copy(), `{}`, 0},
	}
	for _, tt := range tests {
		if got := tt.value.Inspect(); got != tt.expected || tt.value.Len() != tt.length {
			t.Errorf("expected %s with %d keys, got %s with %d", tt.expected, tt.length, got, tt.value.Len())
		}
	}
	if value, ok := copied.Get(key("a")); !ok || value != shared {
		t.Errorf("the copy must share the values, got %v, %t", value, ok)
	}
	if _, ok := copied.Get(key("b")); ok {
		t.Errorf("a deleted key must be gone from the copy")
	}
}
