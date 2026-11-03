package window

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// The automation is tested end to end with a headless window: a program
// runs it on the test's goroutine while a driver sends it commands through
// Drive, the way `lizardscript drive` does.

// driveProgram is a headless app with a canvas surface, hooks and a log
// of what it got.
const driveProgram = `
let w = window.open(40, 30, "app", {"headless": true, "scale": 2, "fps": 240})
let canvas = window.surface(w, "canvas", [10, 10, 20, 10], 1)
let log = []
let greeted = ""
window.automation(w, {
  "dump": function() { {"events": len(log), "greeted": greeted} },
  "stats": function() { {"paint_ms": 0.5} },
  "commands": {
    "Greet": function(args) { greeted = args[0]; [args[0], args[1] + 1] },
    "Fail": function(args) { 1 + "a" },
  },
})
window.run(w, function(events) {
  for (let i = 0; i < len(events); i += 1) {
    let e = events[i]
    if (e.type == "move") {
      push(log, ["move", e.pos, e.pen.pressure, len(e.samples)])
    } elif (e.type == "down" || e.type == "up") {
      push(log, [e.type, e.pos, e.button, e.surface, e.surface_pos, e.mods.shift])
    } elif (e.type == "type") {
      push(log, ["type", e.text])
    } elif (e.type == "key") {
      push(log, ["key", e.key, e.mods.cmd])
    } elif (e.type == "focus") {
      push(log, ["focus", e.focused])
    } elif (e.type == "resize") {
      push(log, ["resize", e.size])
    } else {
      push(log, [e.type])
    }
  }
  window.clear(w, #336699)
  window.clear(canvas, #ff0000)
})
log`

var sprintf = fmt.Sprintf

// drive sends one command and decodes its reply.
func drive(t *testing.T, args ...string) (map[string]any, error) {
	t.Helper()
	var out bytes.Buffer
	err := Drive(args, &out)
	var reply map[string]any
	if jerr := json.Unmarshal(out.Bytes(), &reply); jerr != nil && err == nil {
		t.Fatalf("%v: the reply %q is not JSON: %v", args, out.String(), jerr)
	}
	return reply, err
}

func TestAutomation(t *testing.T) {
	dir := t.TempDir()
	configDir = func() (string, error) { return dir, nil }
	defer func() { configDir = os.UserConfigDir }()

	done := make(chan string)
	go func() {
		// The driver waits for the window's drive file.
		file := DriveFile(filepath.Join(dir, "lizard"), os.Getpid())
		for start := time.Now(); ; time.Sleep(5 * time.Millisecond) {
			if _, err := os.Stat(file); err == nil {
				break
			}
			if time.Since(start) > 10*time.Second {
				done <- "no drive file"
				return
			}
		}
		if info, err := os.Stat(file); err != nil || info.Mode().Perm() != 0o600 {
			done <- "the drive file is not 0600"
			return
		}
		done <- driveSession(t, dir)
	}()
	got := runProgram(t, driveProgram, nil).Inspect()
	if problem := <-done; problem != "" {
		t.Fatal(problem)
	}
	want := `[["down", (15, 12), "left", "canvas", (10, 5), true], ["up", (15, 12), "left", "canvas", (10, 5), true], ` +
		`["down", (0, 0), "left", null, null, false], ["move", (2, 0), 0.5, 1], ["move", (4, 0), 1.0, 1], ["up", (4, 0), "left", null, null, false], ` +
		`["type", "h"], ["type", "i"], ["key", "S", true], ["focus", false], ["leave"], ["resize", (50, 40)], ["close"]]`
	if got != want {
		t.Errorf("the program got\n%s\nwant\n%s", got, want)
	}
	if _, err := os.Stat(DriveFile(filepath.Join(dir, "lizard"), os.Getpid())); !os.IsNotExist(err) {
		t.Errorf("the drive file is still there after the window closed")
	}
}

// driveSession sends every command and checks the replies. It gives what
// went wrong, or "".
func driveSession(t *testing.T, dir string) string {
	var problems []string
	check := func(ok bool, format string, args ...any) {
		if !ok {
			problems = append(problems, sprintf(format, args...))
		}
	}
	list, err := drive(t, "List")
	check(err == nil && strings.Contains(sprintf("%v", list["commands"]), "Screenshot") && sprintf("%v", list["app"]) == "[Fail Greet]",
		"List gave %v, %v", list, err)
	_, err = drive(t, "Click", "15", "12.5", "shift")
	check(err == nil, "Click: %v", err)
	_, err = drive(t, "Drag", "0", "0", "4", "0", "2", "--pressure", "0,1")
	check(err == nil, "Drag: %v", err)
	_, err = drive(t, "Type", "hi")
	check(err == nil, "Type: %v", err)
	_, err = drive(t, "Key", "S", "cmd")
	check(err == nil, "Key: %v", err)
	_, err = drive(t, "Focus", "false")
	check(err == nil, "Focus: %v", err)
	_, err = drive(t, "Leave")
	check(err == nil, "Leave: %v", err)

	shot, err := drive(t, "Screenshot", "--base64")
	check(err == nil && shot["width"] == 80.0 && shot["height"] == 60.0 && len(sprintf("%v", shot["png"])) > 20,
		"Screenshot gave %v, %v", shot, err)
	path := filepath.Join(dir, "shot.png")
	shot, err = drive(t, "Screenshot", path)
	_, serr := os.Stat(path)
	check(err == nil && shot["path"] == path && serr == nil, "Screenshot to a file gave %v, %v", shot, err)

	dump, err := drive(t, "Dump")
	app, _ := dump["app"].(map[string]any)
	check(err == nil && app["events"] == 11.0 && sprintf("%v", dump["surfaces"]) == "[map[name:canvas rect:[10 10 20 10] scale:1 size:[40 20]]]",
		"Dump gave %v, %v", dump, err)
	stats, err := drive(t, "Stats")
	check(err == nil && sprintf("%v", stats["app"]) == "map[paint_ms:0.5]" && stats["ticks"].(float64) > 0, "Stats gave %v, %v", stats, err)

	var out bytes.Buffer
	err = Drive([]string{"Call", "Greet", "liz", "41"}, &out)
	check(err == nil && strings.TrimSpace(out.String()) == `["liz",42]`, "Call Greet gave %s, %v", out.String(), err)
	failed, err := drive(t, "Call", "Fail")
	check(err != nil && strings.Contains(sprintf("%v", failed["error"]), "type mismatch"), "Call Fail gave %v, %v", failed, err)
	missing, err := drive(t, "Call", "Nope")
	check(err != nil && strings.Contains(sprintf("%v", missing["error"]), `the program has no command "Nope"`), "Call Nope gave %v", missing)
	unknown, err := drive(t, "Fly")
	check(err != nil && strings.Contains(sprintf("%v", unknown["error"]), `unknown command "Fly"`), "Fly gave %v", unknown)
	bad, err := drive(t, "Click", "x", "1")
	check(err != nil && strings.Contains(sprintf("%v", bad["error"]), `"x" is not a number`), "a bad Click gave %v", bad)

	_, err = drive(t, "Resize", "50", "40")
	check(err == nil, "Resize: %v", err)
	idle, err := drive(t, "Wait")
	check(err == nil && idle["idle"] == true, "Wait gave %v, %v", idle, err)
	_, err = drive(t, "Quit")
	check(err == nil, "Quit: %v", err)
	return strings.Join(problems, "\n")
}

func TestDriveWithNoWindow(t *testing.T) {
	dir := t.TempDir()
	configDir = func() (string, error) { return dir, nil }
	defer func() { configDir = os.UserConfigDir }()
	os.MkdirAll(filepath.Join(dir, "lizard"), 0o700)
	// The file of a program that ended is left over and removed.
	stale := DriveFile(filepath.Join(dir, "lizard"), 1<<30)
	os.WriteFile(stale, []byte(`{"port": 1}`), 0o600)
	for _, tc := range []struct {
		args []string
		want string
	}{
		{nil, "usage: lizardscript drive"},
		{[]string{"List"}, "no automated window runs"},
		{[]string{"--pid", "x", "List"}, `--pid wants a process id, got "x"`},
		{[]string{"--pid", "99999999", "List"}, "no automated window runs with pid 99999999"},
	} {
		err := Drive(tc.args, &bytes.Buffer{})
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%v gave %v, want %q", tc.args, err, tc.want)
		}
	}
	cleanStale(filepath.Join(dir, "lizard"))
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Errorf("the stale drive file was not removed")
	}
}

func TestAutomationErrors(t *testing.T) {
	for _, tc := range []struct {
		name, source, want string
	}{
		{"not a hash", `window.automation(window.open(1, 1, ""), 5)`, "want true, false or a HASH of hooks, got INTEGER"},
		{"unknown hook", `window.automation(window.open(1, 1, ""), {"paint": 1})`, `unknown hook "paint"`},
		{"dump not a function", `window.automation(window.open(1, 1, ""), {"dump": 1})`, "the hook dump must be a FUNCTION, got INTEGER"},
		{"commands not a hash", `window.automation(window.open(1, 1, ""), {"commands": []})`, "the hook commands must be a HASH of functions, got ARRAY"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := runProgram(t, tc.source, nil)
			if !strings.Contains(got.Inspect(), tc.want) {
				t.Fatalf("got %s, want an error with %q", got.Inspect(), tc.want)
			}
		})
	}
}

func TestSplitArgs(t *testing.T) {
	words, flags := split([]string{"1", "--full", "2", "--pressure", "0.2,1", "--button=right", "shift+cmd"})
	if strings.Join(words, " ") != "1 2 shift+cmd" || len(flags) != 3 || flags["pressure"] != "0.2,1" || flags["button"] != "right" {
		t.Fatalf("got %v %v", words, flags)
	}
	if m := modsOf(words[2:], flags); !m.Shift || !m.Cmd || m.Alt || m.Ctrl {
		t.Errorf("mods %+v, want shift and cmd", m)
	}
}
