package task

import (
	"strings"
	"testing"

	"github.com/alejandruxxug/lizard-script/engine/object"
	"github.com/alejandruxxug/lizard-script/terrarium/internal/arg"
)

func TestWireRoundTrip(t *testing.T) {
	nested := object.NewHash()
	arg.Set(nested, "list", &object.Array{Elements: []object.Object{arg.Int64(1), arg.Float(2.5), arg.Null}})
	nested.Set(&object.Integer{Value: 7}, arg.True)
	nested.Set(arg.False.(*object.Boolean), arg.Str("no"))

	tests := []struct {
		name  string
		value object.Object
	}{
		{"null", arg.Null},
		{"true", arg.True},
		{"false", arg.False},
		{"integer", arg.Int64(-1234567890123)},
		{"float", arg.Float(0.1)},
		{"string", arg.Str("héllo 🦎")},
		{"empty string", arg.Str("")},
		{"color", &object.Color{R: 1, G: 2, B: 3, A: 4}},
		{"point", &object.Point{Coords: []object.Object{arg.Float(1.5), arg.Int64(2), arg.Int64(3)}}},
		{"list", &object.Array{Elements: []object.Object{arg.Str("a"), &object.Array{}}}},
		{"hash", nested},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := encode(nil, tt.value, "args")
			if err != nil {
				t.Fatal(err)
			}
			got, rest, err := decode(data)
			if err != nil {
				t.Fatal(err)
			}
			if len(rest) != 0 {
				t.Errorf("%d bytes left over", len(rest))
			}
			if got.Inspect() != tt.value.Inspect() || got.Type() != tt.value.Type() {
				t.Errorf("got %s %s, want %s %s", got.Type(), got.Inspect(), tt.value.Type(), tt.value.Inspect())
			}
			if _, ok := tt.value.(*object.Hash); ok && got == tt.value {
				t.Error("the hash was shared, not copied")
			}
		})
	}
}

func TestWireRefusesWhatIsNotData(t *testing.T) {
	function := &object.Builtin{Name: "f"}
	inHash := object.NewHash()
	arg.Set(inHash, "draw", function)
	loop := &object.Array{}
	loop.Elements = []object.Object{arg.Int64(1), loop}

	tests := []struct {
		name  string
		value object.Object
		want  string
	}{
		{"a builtin", function, "args is a BUILTIN"},
		{"in a list", &object.Array{Elements: []object.Object{arg.Null, inHash}}, `args[1]["draw"] is a BUILTIN`},
		{"a native", &object.Native{Kind: "image"}, "args is an image"},
		{"a cycle", loop, "args[1] is a list that holds itself"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := encode(nil, tt.value, "args")
			if err == nil || !strings.HasPrefix(err.Error(), tt.want) {
				t.Errorf("got %v, want it to start with %q", err, tt.want)
			}
		})
	}
}

func TestWireBrokenBytes(t *testing.T) {
	data, err := encode(nil, &object.Array{Elements: []object.Object{arg.Str("abc"), arg.Float(1)}}, "args")
	if err != nil {
		t.Fatal(err)
	}
	for cut := 0; cut < len(data); cut++ {
		if _, _, err := decode(data[:cut]); err == nil {
			t.Errorf("cut at %d of %d bytes: no error", cut, len(data))
		}
	}
	// A length far longer than the bytes there asks for no huge list.
	if _, _, err := decode([]byte{tagList, 0xff, 0xff, 0xff, 0xff, 0x0f}); err == nil {
		t.Error("a list longer than its bytes: no error")
	}
}

func TestPrefixWriter(t *testing.T) {
	var out strings.Builder
	w := &prefixWriter{out: &out, prefix: "[task] ", start: true}
	for _, part := range []string{"one\ntw", "o\n", "", "three"} {
		w.Write([]byte(part))
	}
	if want := "[task] one\n[task] two\n[task] three"; out.String() != want {
		t.Errorf("got %q, want %q", out.String(), want)
	}
}
