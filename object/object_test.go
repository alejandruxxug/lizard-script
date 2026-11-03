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
