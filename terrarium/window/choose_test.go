package window

import (
	"reflect"
	"strings"
	"testing"
)

func TestParsePaths(t *testing.T) {
	for _, tc := range []struct {
		name, out string
		want      []string
	}{
		{"cancel", "", []string{}},
		{"only a newline", "\n", []string{}},
		{"one path", "/a/b.png\n", []string{"/a/b.png"}},
		{"many paths", "/a/b.png\n/a/c.jpg\n", []string{"/a/b.png", "/a/c.jpg"}},
		{"spaces", "/my folder/a b.png\n", []string{"/my folder/a b.png"}},
		{"quotes", "/a/it's \"q\".png\n", []string{"/a/it's \"q\".png"}},
		{"windows line ends", `C:\a\b.png` + "\r\n" + `C:\a\c.png` + "\r\n", []string{`C:\a\b.png`, `C:\a\c.png`}},
		{"no final newline", "/a/b.png", []string{"/a/b.png"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := parsePaths(tc.out); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestDialogFor(t *testing.T) {
	all := func(string) bool { return true }
	only := func(tool string) func(string) bool { return func(n string) bool { return n == tool } }
	none := func(string) bool { return false }
	opts := chooseOptions{Title: `Pick "one"`, Types: []string{"png", "jpg"}, Folder: "/my pics", Many: true}

	for _, tc := range []struct {
		name, goos string
		o          chooseOptions
		have       func(string) bool
		program    string
		contains   []string // pieces of the arguments and environment
	}{
		{"mac plain", "darwin", chooseOptions{}, all, "osascript", []string{"set picked to choose file\n"}},
		{"mac everything", "darwin", opts, all, "osascript", []string{
			`choose file with prompt "Pick \"one\"" of type {"png", "jpg"} default location (POSIX file "/my pics") with multiple selections allowed`}},
		{"windows", "windows", opts, all, "powershell", []string{"OpenFileDialog", "LIZARD_TITLE=Pick \"one\"", "LIZARD_FOLDER=/my pics",
			"LIZARD_FILTER=Files (*.png;*.jpg)|*.png;*.jpg|All files|*.*", "LIZARD_MANY=1"}},
		{"windows plain", "windows", chooseOptions{}, all, "powershell", []string{"LIZARD_FILTER=\x00", "LIZARD_MANY=0"}},
		{"zenity", "linux", opts, all, "zenity", []string{"--file-selection", `--title=Pick "one"`, "--filename=/my pics/",
			"--file-filter=Files | *.png *.jpg", "--multiple", "--separator=\n"}},
		{"kdialog when no zenity", "linux", opts, only("kdialog"), "kdialog", []string{"--getopenfilename", "/my pics", "*.png *.jpg", "--title", "--multiple", "--separate-output"}},
		{"kdialog plain", "linux", chooseOptions{}, only("kdialog"), "kdialog", []string{"--getopenfilename", "."}},
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

	if _, err := dialogFor("linux", opts, none); err == nil || !strings.Contains(err.Error(), "zenity or kdialog") {
		t.Fatalf("no tools: got %v", err)
	}
}

func TestChooseFile(t *testing.T) {
	defer func(run func(dialogCommand) (string, error), have func(string) bool) { runDialog, hasTool = run, have }(runDialog, hasTool)
	hasTool = func(string) bool { return true }
	var seen dialogCommand
	for _, tc := range []struct {
		name, out string
		fail      bool
		want      string
	}{
		{"one file", "/a/b c.png\n", false, `["/a/b c.png"]`},
		{"many files", "/a/b.png\n/a/c.png\n", false, `["/a/b.png", "/a/c.png"]`},
		{"cancel", "", true, `[]`},
	} {
		for _, engine := range engines {
			t.Run(tc.name+"/"+engine.name, func(t *testing.T) {
				runDialog = func(c dialogCommand) (string, error) {
					seen = c
					if tc.fail {
						return "", errCancelled
					}
					return tc.out, nil
				}
				got := runOn(t, engine.run, `
let w = window.open(4, 4, "", {"headless": true})
window.choose_file(w, {"title": "Pick", "types": [".png", "jpg"], "folder": "/x", "many": true})
let paths = null
let n = 0
window.run(w, function(events) {
  n += 1
  for (let i = 0; i < len(events); i += 1) {
    if (events[i].type == "chosen") { paths = events[i].paths }
  }
  paths == null && n < 1000000
})
paths`, nil).Inspect()
				if got != tc.want {
					t.Fatalf("got %s, want %s", got, tc.want)
				}
				if !strings.Contains(strings.Join(seen.Args, " "), "Pick") {
					t.Fatalf("the title didn't reach the dialog: %v", seen)
				}
			})
		}
	}
}

type cancelled struct{}

func (cancelled) Error() string { return "exit status 1" }

var errCancelled error = cancelled{}

func TestChooseFileErrors(t *testing.T) {
	defer func(have func(string) bool) { hasTool = have }(hasTool)
	hasTool = func(string) bool { return true }
	for _, tc := range []struct{ name, source, want string }{
		{"unknown option", `window.choose_file(window.open(1, 1, ""), {"kind": 1})`, `unknown option "kind"`},
		{"types not a list", `window.choose_file(window.open(1, 1, ""), {"types": "png"})`, "types must be a list of extensions"},
		{"types not strings", `window.choose_file(window.open(1, 1, ""), {"types": [1]})`, "types must be a list of extensions"},
		{"not a window", `window.choose_file(1)`, "window"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := runProgram(t, tc.source, nil).Inspect(); !strings.Contains(got, tc.want) {
				t.Fatalf("got %s, want it to contain %q", got, tc.want)
			}
		})
	}
}

func TestChooseFileTwice(t *testing.T) {
	defer func(run func(dialogCommand) (string, error), have func(string) bool) { runDialog, hasTool = run, have }(runDialog, hasTool)
	hasTool = func(string) bool { return true }
	release := make(chan struct{})
	runDialog = func(dialogCommand) (string, error) { <-release; return "", nil }
	defer close(release)
	got := runProgram(t, `
let w = window.open(1, 1, "")
window.choose_file(w)
window.choose_file(w)`, nil).Inspect()
	if !strings.Contains(got, "already open") {
		t.Fatalf("got %s", got)
	}
}

func TestInjectChosen(t *testing.T) {
	got := runProgram(t, `
let w = window.open(4, 4, "", {"headless": true})
window.inject(w, {"type": "chosen", "paths": ["/tmp/a.png"]})
let got = null
window.run(w, function(events) { got = events[0]; false })
got`, nil).Inspect()
	if want := `{"type": "chosen", "paths": ["/tmp/a.png"]}`; got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}
