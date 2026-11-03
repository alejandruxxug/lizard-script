package evaluator

import (
	"lizardscript/object"
	"lizardscript/token"
	"strings"
	"testing"
	"time"
)

var testTok = token.Token{Line: 1, Column: 1}

func TestTextFunctions(t *testing.T) {
	ResetMath()
	checkAll(t, []expectation{
		{`text.slice("hello", 1, 3)`, `"el"`},
		{`text.slice("hello", 0, 5)`, `"hello"`},
		{`text.slice("hello", 0, 99)`, `"hello"`},
		{`text.slice("hello", -99, 2)`, `"he"`},
		{`text.slice("hello", -3, -1)`, `"ll"`},
		{`text.slice("hello", 3, 1)`, `""`},
		{`text.slice("hello", 5, 9)`, `""`},
		{`text.slice("", 0, 3)`, `""`},
		{`text.slice("héllo→x", 1, 3)`, `"él"`},
		{`text.slice("héllo→x", -2, 7)`, `"→x"`},

		{`text.find("hello", "l")`, "2"},
		{`text.find("hello", "l", 3)`, "3"},
		{`text.find("hello", "l", 4)`, "-1"},
		{`text.find("hello", "z")`, "-1"},
		{`text.find("hello", "")`, "0"},
		{`text.find("hello", "", 99)`, "5"},
		{`text.find("hello", "h", -99)`, "0"},
		{`text.find("hello", "o", -1)`, "4"},
		{`text.find("", "a")`, "-1"},
		{`text.find("→→a", "a")`, "2"},
		{`text.find("é→é", "é", 1)`, "2"},

		{`text.split("a,b,c", ",")`, `["a", "b", "c"]`},
		{`text.split("a,,c", ",")`, `["a", "", "c"]`},
		{`text.split("abc", "")`, `["a", "b", "c"]`},
		{`text.split("é→", "")`, `["é", "→"]`},
		{`text.split("", ",")`, `[""]`},
		{`text.split("abc", ",")`, `["abc"]`},

		{`text.join(["a", "b"], "-")`, `"a-b"`},
		{`text.join([], "-")`, `""`},
		{`text.join(["a"], "-")`, `"a"`},
		{`text.join(["a", "", "b"], "")`, `"ab"`},

		{`text.replace("a.b.c", ".", "--")`, `"a--b--c"`},
		{`text.replace("aaa", "aa", "b")`, `"ba"`},
		{`text.replace("abc", "x", "y")`, `"abc"`},
		{`text.replace("é→é", "→", "")`, `"éé"`},

		{`text.trim("  hi \n")`, `"hi"`},
		{`text.trim("")`, `""`},
		{`text.trim("   ")`, `""`},

		{`text.lines("a\nb")`, `["a", "b"]`},
		{`text.lines("a" + text.char(13) + "\nb" + text.char(13) + "\n")`, `["a", "b", ""]`},
		{`len(text.lines("a" + text.char(13) + "b")[0])`, "3"},
		{`text.lines("a\n\nb")`, `["a", "", "b"]`},
		{`text.lines("")`, `[""]`},
		{`text.join(text.lines("a" + text.char(13) + "\nb"), "\n")`, "\"a\nb\""},

		{`text.starts("hello", "he")`, "true"},
		{`text.starts("hello", "lo")`, "false"},
		{`text.starts("hello", "")`, "true"},
		{`text.starts("", "a")`, "false"},
		{`text.ends("hello", "lo")`, "true"},
		{`text.ends("hello", "he")`, "false"},
		{`text.ends("hello", "")`, "true"},

		{`text.upper("héllo")`, `"HÉLLO"`},
		{`text.lower("HÉLLO")`, `"héllo"`},
		{`text.upper("")`, `""`},

		{`text.code("A", 0)`, "65"},
		{`text.code("aé→", 1)`, "233"},
		{`text.code("aé→", 2)`, "8594"},
		{`text.code("aé→", -1)`, "8594"},
		{`text.code("abc", 3)`, "null"},
		{`text.code("abc", -4)`, "null"},
		{`text.code("", 0)`, "null"},
		{`text.char(65)`, `"A"`},
		{`text.char(8594)`, `"→"`},
		{`text.char(233)`, `"é"`},
		{`len(text.char(128512))`, "1"},

		{`text.number("42")`, "42"},
		{`text.number("-7")`, "-7"},
		{`text.number("+7")`, "7"},
		{`text.number("  12  ")`, "12"},
		{`text.number("1.5")`, "1.5"},
		{`text.number(".5")`, "0.5"},
		{`text.number("5.")`, "5.0"},
		{`text.number("1e3")`, "1000.0"},
		{`text.number("-2.5E-1")`, "-0.25"},
		{`type(text.number("12"))`, `"INTEGER"`},
		{`type(text.number("12.0"))`, `"FLOAT"`},
		{`type(text.number("99999999999999999999"))`, `"FLOAT"`},
		{`text.number("")`, "null"},
		{`text.number("abc")`, "null"},
		{`text.number("12px")`, "null"},
		{`text.number("1e999")`, "null"},
		{`text.number("Inf")`, "null"},
		{`text.number("NaN")`, "null"},
		{`text.number("0x10")`, "null"},
		{`text.number("1_000")`, "null"},
		{`text.number("1 2")`, "null"},
		{`text.number("-")`, "null"},
		{`text.number("١٢")`, "null"},

		{`let text = 1; text`, "1"},
		{`len(keys(text))`, "14"},
	})
}

func TestTextErrors(t *testing.T) {
	ResetMath()
	checkErrors(t, []expectation{
		{`text.slice("a", 0)`, "[line 1, column 11] runtime error: text.slice: wrong number of arguments: want 3, got 2"},
		{`text.slice(1, 0, 1)`, "[line 1, column 11] runtime error: text.slice: argument 1 must be STRING, got INTEGER"},
		{`text.slice("a", "0", 1)`, "[line 1, column 11] runtime error: text.slice: argument 2 must be INTEGER, got STRING"},
		{`text.slice("a", 0.5, 1)`, "[line 1, column 11] runtime error: text.slice: argument 2 must be INTEGER, got FLOAT"},
		{`text.find("a")`, "[line 1, column 10] runtime error: text.find: wrong number of arguments: want 2 to 3, got 1"},
		{`text.find("a", "a", "b")`, "[line 1, column 10] runtime error: text.find: argument 3 must be INTEGER, got STRING"},
		{`text.join("a", ",")`, "[line 1, column 10] runtime error: text.join: argument 1 must be ARRAY, got STRING"},
		{`text.join([1], ",")`, "[line 1, column 10] runtime error: text.join: element 0 of the list must be STRING, got INTEGER"},
		{`text.replace("a", "", "b")`, "[line 1, column 13] runtime error: text.replace: the text to replace can't be empty"},
		{`text.trim()`, "[line 1, column 10] runtime error: text.trim: wrong number of arguments: want 1, got 0"},
		{`text.upper(1)`, "[line 1, column 11] runtime error: text.upper: argument 1 must be STRING, got INTEGER"},
		{`text.number(5)`, "[line 1, column 12] runtime error: text.number: argument 1 must be STRING, got INTEGER"},
		{`text.char(-1)`, "[line 1, column 10] runtime error: text.char: -1 is not a character code"},
		{`text.char(1114112)`, "[line 1, column 10] runtime error: text.char: 1114112 is not a character code"},
		{`text.char(55296)`, "[line 1, column 10] runtime error: text.char: 55296 is not a character code"},
		{`text.char("a")`, "[line 1, column 10] runtime error: text.char: argument 1 must be INTEGER, got STRING"},
	})
}

// TestStringIndexCache checks that the cached characters give the same
// answers as the plain conversion, for ASCII and non-ASCII text, and that
// the helpers can be mixed on the same string.
func TestStringIndexCache(t *testing.T) {
	checkAll(t, []expectation{
		{`let s = "añ🦎b"; [len(s), s[0], s[1], s[2], s[3], s[4], s[-1]]`, `[4, "a", "ñ", "🦎", "b", null, null]`},
		{`let s = "añ🦎b"; [text.slice(s, 1, 3), text.code(s, 2), text.find(s, "b"), len(s)]`, `["ñ🦎", 129422, 3, 4]`},
		{`let s = "hello"; [len(s), s[4], text.slice(s, -3, 5), text.code(s, -1), text.find(s, "l", 3)]`, `[5, "o", "llo", 111, 3]`},
		{`let s = ""; [len(s), s[0], text.slice(s, 0, 3), text.find(s, "")]`, `[0, null, "", 0]`},
		{`"\u00e9\u{1F98E}"[1]`, `"🦎"`},
	})
}

// longString is n characters of text; wide mixes in a non-ASCII one.
func longString(n int, wide bool) *object.String {
	b := []rune(strings.Repeat("abcdefgh", n/8+1))[:n]
	if wide {
		b[0] = 'é'
	}
	return &object.String{Value: string(b)}
}

// BenchmarkIndexEveryChar reads every character of a 200 KB string by index,
// the loop that used to be O(n squared).
func BenchmarkIndexEveryChar(b *testing.B) {
	for _, wide := range []bool{false, true} {
		name := map[bool]string{false: "ascii", true: "unicode"}[wide]
		b.Run(name, func(b *testing.B) {
			s := longString(200_000, wide)
			n := int64(s.Len())
			for i := 0; i < b.N; i++ {
				for j := int64(0); j < n; j++ {
					evalIndexExpression(testTok, s, &object.Integer{Value: j})
				}
			}
		})
	}
}

// BenchmarkIndexEveryCharOld is the same loop with the old conversion on
// every access, for comparison.
func BenchmarkIndexEveryCharOld(b *testing.B) {
	s := longString(20_000, false) // 10 times smaller: the old loop is too slow at 200 KB
	n := len([]rune(s.Value))
	for i := 0; i < b.N; i++ {
		for j := 0; j < n; j++ {
			_ = string([]rune(s.Value)[j])
		}
	}
}

// TestIndexIsLinear fails if indexing every character of a long string goes
// back to converting the whole string on every access.
func TestIndexIsLinear(t *testing.T) {
	s := longString(200_000, true)
	start := time.Now()
	for j := 0; j < s.Len(); j++ {
		evalIndexExpression(testTok, s, &object.Integer{Value: int64(j)})
	}
	if took := time.Since(start); took > 5*time.Second {
		t.Errorf("indexing 200000 characters took %v", took)
	}
}
