package terrarium

import "testing"

func TestJSONParse(t *testing.T) {
	check(t, []testCase{
		{"integer", `json.parse("42")`, "42", false},
		{"negative float", `json.parse("-1.5e2")`, "-150.0", false},
		{"too big for an integer", `type(json.parse("12345678901234567890"))`, "FLOAT", false},
		{"words", `json.parse("[true, false, null]")`, "[true, false, null]", false},
		{"string", `json.parse("\"liz\"")`, "liz", false},
		{"escapes", `json.parse("\"a\\\"b\\\\c\\/d\\ne\\tf\"")`, "a\"b\\c/d\ne\tf", false},
		{"unicode escape", `json.parse("\"\\u00e9\\u0041\"")`, "éA", false},
		{"surrogate pair", `json.parse("\"\\ud83e\\udd8e\"")`, "🦎", false},
		{"nested", `json.parse("{\"a\": [1, {\"b\": [2, []]}], \"c\": {}}")`, `{"a": [1, {"b": [2, []]}], "c": {}}`, false},
		{"whitespace", "json.parse(\" \\r\\n\\t[ 1 ,2 ] \")", "[1, 2]", false},
		{"key order kept", `keys(json.parse("{\"z\": 1, \"a\": 2}"))`, `["z", "a"]`, false},

		{"not a string", `json.parse(5)`, "json.parse: want a STRING text, got INTEGER", true},
		{"empty", `json.parse("")`, "line 1, column 1: the text ends where a value should be", true},
		{"trailing", `json.parse("1 2")`, "line 1, column 3: unexpected '2' after the end of the value", true},
		{"place on later line", `json.parse("[1,\n  x]")`, "line 2, column 3: unexpected 'x'", true},
		{"open string", `json.parse("\"abc")`, "this string is never closed", true},
		{"bad escape", `json.parse("\"\\q\"")`, "bad escape \\q", true},
		{"lone low surrogate", `json.parse("\"\\udc00\"")`, "a low surrogate", true},
		{"lone high surrogate", `json.parse("\"\\ud800x\"")`, "a high surrogate must be followed", true},
		{"control in string", "json.parse(\"\\\"a\\tb\\\"\")", "write it as an escape", true},
		{"leading zero", `json.parse("01")`, "extra zeros", true},
		{"missing colon", `json.parse("{\"a\" 1}")`, "expected : after the key \"a\"", true},
		{"missing comma", `json.parse("[1 2]")`, "expected , or ] in an array, got '2'", true},
		{"unclosed object", `json.parse("{\"a\": 1")`, "the object is never closed", true},
		{"bad word", `json.parse("nul")`, "expected null, found 'nul'", true},
		{"too deep", `json.parse("[" * 102 + "]" * 102)`, "nested more than 100 levels deep", true},
		{"catchable", `let m = ""; try { json.parse("[") } catch (e) { m = e.message }; text.starts(m, "json.parse: line 1, column 2")`, "true", false},
	})
}

func TestJSONWrite(t *testing.T) {
	check(t, []testCase{
		{"scalars", `json.write([1, 2.5, true, false, null, "x"])`, `[1,2.5,true,false,null,"x"]`, false},
		{"escapes", `json.write("a\"b\\c\nd\te" + text.char(1))`, `"a\"b\\c\nd\te\u0001"`, false},
		{"unicode stays", `json.write("é🦎")`, `"é🦎"`, false},
		{"color and point", `json.write([#ff8800, (1, 2), (1, 2.5, 3)])`, `["#ff8800",[1,2],[1,2.5,3]]`, false},
		{"hash", `json.write({"a": 1, 2: [3]})`, `{"a":1,"2":[3]}`, false},
		{"empty", `json.write([[], {}])`, `[[],{}]`, false},
		{"indent", `json.write({"a": [1, 2], "b": {}}, 2)`, "{\n  \"a\": [\n    1,\n    2\n  ],\n  \"b\": {}\n}", false},
		{"indent 0 is compact", `json.write({"a": 1}, 0)`, `{"a":1}`, false},

		{"function", `json.write(function() {})`, "json.write: can't write a FUNCTION as JSON", true},
		{"contains itself", `let a = []; push(a, a); json.write(a)`, "nested more than 100 levels deep", true},
		{"bad indent", `json.write(1, -1)`, "the indent must be an INTEGER from 0 to 16", true},
	})
}

func TestJSONRoundTrip(t *testing.T) {
	values := []string{
		`{"name": "Liz", "toes": [5, 5, 5, 5], "tail": {"long": true, "scales": null}}`,
		`["", "line\nbreak", "quote \" and \\ slash", "tab\t", "é 🦎"]`,
		`[[[[1]]], {"a": {"b": {"c": -0.25}}}, 10.0 ** 21, 0.000001 / 1000]`,
	}
	for _, v := range values {
		for _, indent := range []string{"0", "4"} {
			source := "let v = " + v + "\n" +
				"let back = json.parse(json.write(v, " + indent + "));\n" +
				"[v == back, json.write(back) == json.write(v)]"
			if got := run(t, source).Inspect(); got != "[true, true]" {
				t.Errorf("round trip of %s with indent %s gave %s", v, indent, got)
			}
		}
	}
}
