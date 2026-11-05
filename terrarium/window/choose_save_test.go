package window

import (
	"github.com/alejandruxxug/lizard-script/engine/object"
	"github.com/alejandruxxug/lizard-script/lang/ast"
	"reflect"
	"strings"
	"testing"
)

func TestSaveDialogFor(t *testing.T) {
	all := func(string) bool { return true }
	only := func(tool string) func(string) bool { return func(n string) bool { return n == tool } }
	opts := chooseOptions{Save: true, Title: `Save "it"`, Name: "a b.png", Types: []string{"png"}, Folder: "/my pics/"}
	plain := chooseOptions{Save: true}

	for _, tc := range []struct {
		name, goos string
		o          chooseOptions
		have       func(string) bool
		program    string
		contains   []string
	}{
		{"mac", "darwin", opts, all, "osascript", []string{
			`choose file name with prompt "Save \"it\"" default name "a b.png" default location (POSIX file "/my pics/")`, "POSIX path of picked"}},
		{"mac plain", "darwin", plain, all, "osascript", []string{"set picked to choose file name\n"}},
		{"windows", "windows", opts, all, "powershell", []string{"SaveFileDialog", "OverwritePrompt", "LIZARD_NAME=a b.png",
			`LIZARD_TITLE=Save "it"`, "LIZARD_FOLDER=/my pics/", "LIZARD_EXT=png", "LIZARD_FILTER=Files (*.png)|*.png|All files|*.*"}},
		{"windows plain", "windows", plain, all, "powershell", []string{"LIZARD_NAME=\x00", "LIZARD_EXT="}},
		{"zenity", "linux", opts, all, "zenity", []string{"--file-selection", "--save", "--confirm-overwrite",
			`--title=Save "it"`, "--filename=/my pics/a b.png", "--file-filter=Files | *.png"}},
		{"zenity plain", "linux", plain, all, "zenity", []string{"--save", "--confirm-overwrite"}},
		{"kdialog", "linux", opts, only("kdialog"), "kdialog", []string{"--getsavefilename", "/my pics/a b.png", "*.png", "--title"}},
		{"kdialog plain", "linux", plain, only("kdialog"), "kdialog", []string{"--getsavefilename", "./"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd, err := dialogFor(tc.goos, tc.o, tc.have)
			if err != nil {
				t.Fatal(err)
			}
			if cmd.Name != tc.program {
				t.Fatalf("program %q, want %q", cmd.Name, tc.program)
			}
			all := strings.Join(cmd.Args, "\x00") + "\x00" + strings.Join(cmd.Env, "\x00")
			for _, piece := range tc.contains {
				if !strings.Contains(all, piece) {
					t.Errorf("%q is not in %q", piece, all)
				}
			}
		})
	}
	if _, err := dialogFor("linux", plain, func(string) bool { return false }); err == nil || !strings.Contains(err.Error(), "choose_save needs zenity") {
		t.Fatalf("no tools: got %v", err)
	}
}

func TestSavedPath(t *testing.T) {
	png := []string{"png", "jpg"}
	for _, tc := range []struct {
		name, goos string
		paths      []string
		types      []string
		want       []string
	}{
		{"cancel", "darwin", []string{}, png, []string{}},
		{"mac adds the extension", "darwin", []string{"/a/pic"}, png, []string{"/a/pic.png"}},
		{"mac keeps an extension", "darwin", []string{"/a/pic.jpg"}, png, []string{"/a/pic.jpg"}},
		{"mac no types", "darwin", []string{"/a/pic"}, nil, []string{"/a/pic"}},
		{"linux leaves it", "linux", []string{"/a/pic"}, png, []string{"/a/pic"}},
		{"only the first path", "linux", []string{"/a/x", "/a/y"}, nil, []string{"/a/x"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := savedPath(tc.goos, tc.paths, tc.types); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// answer runs a program that opens a dialog with the call, then gives back
// the first chosen event it sees.
func answer(t *testing.T, engine func(*ast.Program, *object.Environment) object.Object, call string) string {
	return runOn(t, engine, `
let w = window.open(4, 4, "", {"headless": true})
`+call+`
let got = null
let n = 0
window.run(w, function(events) {
  n += 1
  for (let i = 0; i < len(events); i += 1) {
    if (events[i].type == "chosen") { got = events[i] }
  }
  got == null && n < 1000000
})
got`, nil).Inspect()
}

func TestChooseSave(t *testing.T) {
	defer func(run func(dialogCommand) (string, error), have func(string) bool, os string) {
		runDialog, hasTool, dialogOS = run, have, os
	}(runDialog, hasTool, dialogOS)
	hasTool = func(string) bool { return true }
	for _, tc := range []struct {
		name, os, out, call string
		fail                bool
		want                string
	}{
		{"path", "linux", "/a/b.png\n", `window.choose_save(w, {"name": "b.png"})`,
			false, `{"type": "chosen", "paths": ["/a/b.png"], "dialog": "save"}`},
		{"id echoed", "linux", "/a/b.png\n", `window.choose_save(w, {"id": "export"})`,
			false, `{"type": "chosen", "paths": ["/a/b.png"], "dialog": "save", "id": "export"}`},
		{"mac extension", "darwin", "/a/b\n", `window.choose_save(w, {"types": ["png"]})`,
			false, `{"type": "chosen", "paths": ["/a/b.png"], "dialog": "save"}`},
		{"cancel", "linux", "", `window.choose_save(w, {"id": "x"})`,
			true, `{"type": "chosen", "paths": [], "dialog": "save", "id": "x"}`},
		{"open says open", "linux", "/a/b.png\n", `window.choose_file(w, {"id": "pick"})`,
			false, `{"type": "chosen", "paths": ["/a/b.png"], "dialog": "open", "id": "pick"}`},
	} {
		for _, engine := range engines {
			t.Run(tc.name+"/"+engine.name, func(t *testing.T) {
				dialogOS = tc.os
				runDialog = func(dialogCommand) (string, error) {
					if tc.fail {
						return "", errCancelled
					}
					return tc.out, nil
				}
				if got := answer(t, engine.run, tc.call); got != tc.want {
					t.Fatalf("got %s, want %s", got, tc.want)
				}
			})
		}
	}
}

func TestChooseSaveErrors(t *testing.T) {
	defer func(have func(string) bool) { hasTool = have }(hasTool)
	hasTool = func(string) bool { return true }
	for _, tc := range []struct{ name, source, want string }{
		{"unknown option", `window.choose_save(window.open(1, 1, ""), {"many": true})`, `unknown option "many"`},
		{"not a window", `window.choose_save(1)`, "window"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := runProgram(t, tc.source, nil).Inspect(); !strings.Contains(got, tc.want) {
				t.Fatalf("got %s, want it to contain %q", got, tc.want)
			}
		})
	}
}

func TestChooseSaveWhileOpen(t *testing.T) {
	defer func(run func(dialogCommand) (string, error), have func(string) bool) { runDialog, hasTool = run, have }(runDialog, hasTool)
	hasTool = func(string) bool { return true }
	release := make(chan struct{})
	runDialog = func(dialogCommand) (string, error) { <-release; return "", nil }
	defer close(release)
	got := runProgram(t, `
let w = window.open(1, 1, "")
window.choose_file(w)
window.choose_save(w)`, nil).Inspect()
	if !strings.Contains(got, "already open") {
		t.Fatalf("got %s", got)
	}
}

func TestInjectChosenSave(t *testing.T) {
	got := runProgram(t, `
let w = window.open(4, 4, "", {"headless": true})
window.inject(w, {"type": "chosen", "paths": ["/tmp/a.png"], "dialog": "save", "id": "e"})
let got = null
window.run(w, function(events) { got = events[0]; false })
got`, nil).Inspect()
	if want := `{"type": "chosen", "paths": ["/tmp/a.png"], "dialog": "save", "id": "e"}`; got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}
