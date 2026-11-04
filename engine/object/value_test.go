package object

import (
	"hash/fnv"
	"math"
	"strings"
	"testing"
)

func TestValues(t *testing.T) {
	str := &String{Value: "s"}
	tests := []struct {
		name   string
		object Object
		kind   ValueKind
	}{
		{"small integer", IntegerOf(5), KindInt},
		{"big integer", &Integer{Value: 1 << 40}, KindInt},
		{"negative integer", &Integer{Value: -7}, KindInt},
		{"float", &Float{Value: 2.5}, KindFloat},
		{"minus zero", &Float{Value: math.Copysign(0, -1)}, KindFloat},
		{"string", str, KindObject},
		{"boolean", &Boolean{Value: true}, KindObject},
	}
	for _, tt := range tests {
		v := ValueOf(tt.object)
		if v.Kind() != tt.kind {
			t.Errorf("%s: kind %d, want %d", tt.name, v.Kind(), tt.kind)
		}
		back := v.Object()
		if back.Type() != tt.object.Type() || back.Inspect() != tt.object.Inspect() {
			t.Errorf("%s: boxed back as %s %s, want %s %s", tt.name, back.Type(), back.Inspect(), tt.object.Type(), tt.object.Inspect())
		}
		if f, ok := tt.object.(*Float); ok && math.Signbit(back.(*Float).Value) != math.Signbit(f.Value) {
			t.Errorf("%s: lost its sign", tt.name)
		}
	}
	if ValueOf(str).Object() != str || ValueOf(str).Ref() != str {
		t.Error("an object must come back as itself")
	}
	if !(Value{}).IsEmpty() || IntValue(0).IsEmpty() || ValueOf(NewHash()).IsEmpty() {
		t.Error("only the zero Value is empty")
	}
	if IntValue(3).Number() != 3 || FloatValue(0.5).Number() != 0.5 || IntValue(-2).Int() != -2 {
		t.Error("numbers read back wrong")
	}
}

func TestConcat(t *testing.T) {
	// Built piece by piece, the string is right at every step, and the
	// earlier strings keep their own text.
	s := &String{Value: ""}
	var steps []*String
	for i := 0; i < 200; i++ {
		s = Concat(s, &String{Value: "ab"})
		steps = append(steps, s)
	}
	for i, step := range steps {
		if want := strings.Repeat("ab", i+1); step.Value != want {
			t.Fatalf("step %d: %q", i, step.Value)
		}
	}

	// Two strings made from the same one don't see each other's bytes.
	base := Concat(&String{Value: strings.Repeat("q", 70)}, &String{Value: "w"})
	a := Concat(base, &String{Value: "1"})
	b := Concat(base, &String{Value: "2"})
	c := Concat(a, &String{Value: "3"})
	prefix := strings.Repeat("q", 70) + "w"
	if base.Value != prefix || a.Value != prefix+"1" || b.Value != prefix+"2" || c.Value != prefix+"13" {
		t.Errorf("shared buffers mixed up: %q %q %q %q", base.Value, a.Value, b.Value, c.Value)
	}

	// The length in characters stays right across ASCII and non-ASCII pieces.
	ascii := &String{Value: strings.Repeat("x", 64)}
	ascii.Len()
	mixed := Concat(Concat(ascii, &String{Value: "ñ"}), &String{Value: "y"})
	if mixed.Len() != 66 || mixed.RuneAt(64) != 'ñ' || mixed.RuneAt(65) != 'y' {
		t.Errorf("mixed string: len %d", mixed.Len())
	}
	if joined := Concat(ascii, &String{Value: "z"}); joined.KnownLen() != 65 {
		t.Errorf("an ASCII string joined with ASCII knows its length: %d", joined.KnownLen())
	}
	if (&String{Value: "abc"}).KnownLen() != -1 {
		t.Error("a string not looked at yet doesn't know its length")
	}
}

func TestChar(t *testing.T) {
	if Char('a') != Char('a') || Char('a').Value != "a" || Char('a').Len() != 1 || Char('a').RuneAt(0) != 'a' {
		t.Error("ASCII characters are shared and read right")
	}
	if c := Char('ñ'); c.Value != "ñ" || c.Len() != 1 || c == Char('ñ') {
		t.Error("other characters are new strings")
	}
}

func TestStringHashKey(t *testing.T) {
	for _, text := range []string{"", "a", "id", "ñandú 🦎", strings.Repeat("xyz", 50)} {
		h := fnv.New64a()
		h.Write([]byte(text))
		if got := (&String{Value: text}).HashKey(); got.Value != h.Sum64() || got.Type != STRING_OBJ {
			t.Errorf("%q: hash key %v, want FNV-1a %d", text, got, h.Sum64())
		}
	}
}

func TestSmallHashesGrow(t *testing.T) {
	h := NewHashSized(2)
	for i := 0; i < 10; i++ {
		h.Set(IntegerOf(int64(i)), IntegerOf(int64(i*i)))
	}
	if h.Len() != 10 || len(h.Order) != 10 {
		t.Fatalf("len %d, order %d", h.Len(), len(h.Order))
	}
	for i, key := range h.Order {
		if pair := h.Pairs[key]; pair.Key.Inspect() != IntegerOf(int64(i)).Inspect() {
			t.Errorf("key %d is %s", i, pair.Key.Inspect())
		}
	}
}

func TestOwnVersion(t *testing.T) {
	outer := NewEnvironment()
	inner := NewEnclosedEnvironment(outer)
	outer.Define("a", IntegerOf(1))
	if version, all := outer.OwnVersion(); !all || version != outer.Version() {
		t.Errorf("an environment alone: own version %d, all %t, version %d", version, all, outer.Version())
	}
	if _, all := inner.OwnVersion(); all {
		t.Error("an environment inside another doesn't have all of its version")
	}
}
