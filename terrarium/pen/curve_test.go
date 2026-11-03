package pen

import (
	"lizardscript/ast"
	"lizardscript/evaluator"
	"lizardscript/lexer"
	"lizardscript/object"
	"lizardscript/parser"
	"lizardscript/vm"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// engines are both engines, which must give the same for every program.
var engines = []struct {
	name string
	run  Runner
}{
	{"evaluator", func(p *ast.Program, env *object.Environment) object.Object { return evaluator.Eval(p, env) }},
	{"vm", vm.Run},
}

// resetCurves puts the default curves back when the test ends.
func resetCurves(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		mu.Lock()
		curves = defaultCurves()
		mu.Unlock()
	})
}

func TestCurveApply(t *testing.T) {
	trackpad := Curve{Low: 0.8, High: 1, Gamma: 1.5}
	for _, tc := range []struct {
		name  string
		curve Curve
		raw   float64
		want  float64
	}{
		{"identity", Identity, 0.37, 0.37},
		{"below low", trackpad, 0.5, 0},
		{"at low", trackpad, 0.8, 0},
		{"halfway, bent", trackpad, 0.9, math.Pow(0.5, 1.5)},
		{"at high", trackpad, 1, 1},
		{"above high", Curve{Low: 0, High: 0.5, Gamma: 1}, 0.75, 1},
		{"gamma below 1 raises", Curve{Low: 0, High: 1, Gamma: 0.5}, 0.25, 0.5},
		{"negative raw", Identity, -0.2, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.curve.Apply(tc.raw); math.Abs(got-tc.want) > 1e-12 {
				t.Errorf("Apply(%v) = %v, want %v", tc.raw, got, tc.want)
			}
		})
	}
}

func TestCurveValidate(t *testing.T) {
	for _, tc := range []struct {
		name  string
		curve Curve
		err   string // empty: valid
	}{
		{"identity", Identity, ""},
		{"edges", Curve{Low: 0, High: 1, Gamma: 10}, ""},
		{"negative low", Curve{Low: -0.1, High: 1, Gamma: 1}, "low must be"},
		{"low is 1", Curve{Low: 1, High: 1, Gamma: 1}, "low must be"},
		{"high not above low", Curve{Low: 0.5, High: 0.5, Gamma: 1}, "high must be above low"},
		{"high past 1", Curve{Low: 0, High: 1.2, Gamma: 1}, "high must be above low"},
		{"gamma too small", Curve{Low: 0, High: 1, Gamma: 0.05}, "gamma must be"},
		{"gamma too big", Curve{Low: 0, High: 1, Gamma: 11}, "gamma must be"},
		{"NaN", Curve{Low: math.NaN(), High: 1, Gamma: 1}, "low must be"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			checkErr(t, tc.curve.Validate(), tc.err)
		})
	}
}

// checkErr fails unless err contains want (or is nil when want is empty).
func checkErr(t *testing.T, err error, want string) {
	t.Helper()
	switch {
	case want == "" && err != nil:
		t.Errorf("unexpected error: %v", err)
	case want != "" && err == nil:
		t.Errorf("want an error with %q, got none", want)
	case want != "" && !strings.Contains(err.Error(), want):
		t.Errorf("error %q doesn't contain %q", err, want)
	}
}

func TestDefaultCurves(t *testing.T) {
	d := defaultCurves()
	if d["tablet"] != Identity || d["mouse"] != Identity {
		t.Errorf("tablet and mouse should be the identity, got %+v", d)
	}
	if want := (Curve{Low: 0.8, High: 1, Gamma: 1.5}); d["trackpad"] != want {
		t.Errorf("trackpad = %+v, want %+v", d["trackpad"], want)
	}
	for source, c := range d {
		if err := c.Validate(); err != nil {
			t.Errorf("default %s is invalid: %v", source, err)
		}
	}
}

func TestSetCurveAndShape(t *testing.T) {
	resetCurves(t)
	checkErr(t, SetCurve("tablet", Curve{Low: 0.2, High: 1, Gamma: 1}), "")
	if got := Shape("tablet", 0.6); math.Abs(got-0.5) > 1e-12 {
		t.Errorf("Shape = %v, want 0.5", got)
	}
	checkErr(t, SetCurve("stylus", Identity), `unknown source "stylus"`)
	checkErr(t, SetCurve("tablet", Curve{Low: 0.9, High: 0.1, Gamma: 1}), "high must be")
	if got := CurveOf("tablet").Low; got != 0.2 {
		t.Errorf("a rejected curve changed the tablet: low = %v", got)
	}
	if got := Shape("unknown", 0.4); got != 0.4 {
		t.Errorf("an unknown source should pass through, got %v", got)
	}
}

// runPen runs source with the pen module as pen, on the given engine.
func runPen(t *testing.T, run Runner, source string) object.Object {
	t.Helper()
	p := parser.New(lexer.New(source))
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		t.Fatalf("parse errors: %v", p.Errors())
	}
	env := object.NewEnvironment()
	env.Define("pen", Module())
	return run(program, env)
}

func TestCurveFunction(t *testing.T) {
	for _, e := range engines {
		for _, tc := range []struct {
			name, source string
			want         string // the value as Inspect shows it, or part of the error
			err          bool
		}{
			{"get default", `pen.curve("tablet")`, `{"low": 0.0, "high": 1.0, "gamma": 1.0}`, false},
			{"get trackpad", `pen.curve("trackpad")`, `{"low": 0.8, "high": 1.0, "gamma": 1.5}`, false},
			{"set returns the new curve", `pen.curve("mouse", {"gamma": 2})`, `{"low": 0.0, "high": 1.0, "gamma": 2.0}`, false},
			{"missing keys keep their value", `pen.curve("trackpad", {"low": 0.5}); pen.curve("trackpad")`, `{"low": 0.5, "high": 1.0, "gamma": 1.5}`, false},
			{"unknown source", `pen.curve("stylus")`, `unknown source "stylus"`, true},
			{"unknown key", `pen.curve("mouse", {"lo": 0.1})`, `unknown key "lo"`, true},
			{"not a number", `pen.curve("mouse", {"low": "a"})`, "low must be a number, got STRING", true},
			{"low too high", `pen.curve("mouse", {"low": 1})`, "low must be", true},
			{"gamma out of range", `pen.curve("mouse", {"gamma": 20})`, "gamma must be from 0.1 to 10", true},
			{"high below low", `pen.curve("mouse", {"high": 0})`, "high must be above low", true},
			{"no arguments", `pen.curve()`, "wrong number of arguments", true},
			{"not a hash", `pen.curve("mouse", 3)`, "HASH", true},
		} {
			t.Run(e.name+"/"+tc.name, func(t *testing.T) {
				resetCurves(t)
				got := runPen(t, e.run, tc.source)
				if err, isErr := got.(*object.Error); isErr {
					if !tc.err || !strings.Contains(err.Message, tc.want) {
						t.Fatalf("error %q, want err=%v containing %q", err.Message, tc.err, tc.want)
					}
					return
				}
				if tc.err || got.Inspect() != tc.want {
					t.Errorf("got %s, want %s (error: %v)", got.Inspect(), tc.want, tc.err)
				}
			})
		}
	}
}

func TestApplyConfig(t *testing.T) {
	for _, e := range engines {
		for _, tc := range []struct {
			name, code string
			trackpad   Curve  // the trackpad curve afterwards
			err        string // empty: the file loads
		}{
			{"empty file", ``, defaultCurves()["trackpad"], ""},
			{"partial hash keeps the rest", `let trackpad = {"gamma": 2}`, Curve{0.8, 1, 2}, ""},
			{"computed values", `let low = 0.7
let trackpad = {"low": low, "high": 0.95, "gamma": 1}`, Curve{0.7, 0.95, 1}, ""},
			{"other variables are ignored", `let speed = 3`, defaultCurves()["trackpad"], ""},
			{"syntax error", `let trackpad = `, defaultCurves()["trackpad"], "syntax error"},
			{"runtime error", `let trackpad = nothing_here`, defaultCurves()["trackpad"], "identifier not found"},
			{"not a hash", `let trackpad = 3`, defaultCurves()["trackpad"], "trackpad must be a hash"},
			{"invalid value", `let trackpad = {"gamma": 99}`, defaultCurves()["trackpad"], "trackpad: gamma must be"},
			{"one bad curve sets none", `let trackpad = {"gamma": 2}
let tablet = {"low": 2}`, defaultCurves()["trackpad"], "tablet: low must be"},
		} {
			t.Run(e.name+"/"+tc.name, func(t *testing.T) {
				resetCurves(t)
				checkErr(t, applyConfig(tc.code, "pen.liz", e.run), tc.err)
				if got := CurveOf("trackpad"); got != tc.trackpad {
					t.Errorf("trackpad = %+v, want %+v", got, tc.trackpad)
				}
			})
		}
	}
}

func TestLoadConfigFile(t *testing.T) {
	resetCurves(t)
	dir := t.TempDir()
	checkErr(t, LoadConfig(filepath.Join(dir, "missing.liz"), engines[0].run), "")

	path := filepath.Join(dir, "pen.liz")
	if err := os.WriteFile(path, []byte(`let tablet = {"low": 0.1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	checkErr(t, LoadConfig(path, engines[0].run), "")
	if got := CurveOf("tablet").Low; got != 0.1 {
		t.Errorf("tablet low = %v, want 0.1", got)
	}
}

func TestConfigPath(t *testing.T) {
	t.Setenv("HOME", "/home/someone")
	t.Setenv("XDG_CONFIG_HOME", "/home/someone/.config")
	path, err := ConfigPath()
	if err != nil {
		t.Skip("no config dir on this system")
	}
	if filepath.Base(path) != "pen.liz" || filepath.Base(filepath.Dir(path)) != "lizard" {
		t.Errorf("ConfigPath = %s, want .../lizard/pen.liz", path)
	}
}
