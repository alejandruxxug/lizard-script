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

// runDriven runs program on engine while session drives it from another
// goroutine, and gives what the program returned and what the session found
// wrong.
func runDriven(t *testing.T, engine int, program string, session func(dir string) string) (string, string) {
	t.Helper()
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
		done <- session(dir)
	}()
	got := runOn(t, engines[engine].run, program, nil).Inspect()
	problem := <-done
	if _, err := os.Stat(DriveFile(filepath.Join(dir, "lizard"), os.Getpid())); !os.IsNotExist(err) {
		problem += "\nthe drive file is still there after the window closed"
	}
	return got, problem
}

func TestAutomation(t *testing.T) {
	want := `[["move", (15, 12), 1.0, 1], ["down", (15, 12), "left", "canvas", (10, 5), true], ["up", (15, 12), "left", "canvas", (10, 5), true], ` +
		`["down", (0, 0), "left", null, null, false], ["move", (2, 0), 0.5, 1], ["move", (4, 0), 1.0, 1], ["up", (4, 0), "left", null, null, false], ` +
		`["type", "h"], ["type", "i"], ["key", "S", true], ["focus", false], ["leave"], ["resize", (50, 40)], ["close"]]`
	for i, engine := range engines {
		t.Run(engine.name, func(t *testing.T) {
			got, problem := runDriven(t, i, driveProgram, func(dir string) string { return driveSession(t, dir) })
			if problem != "" {
				t.Fatal(problem)
			}
			if got != want {
				t.Errorf("the program got\n%s\nwant\n%s", got, want)
			}
		})
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
	check(err == nil && app["events"] == 12.0 && sprintf("%v", dump["surfaces"]) == "[map[name:canvas rect:[10 10 20 10] scale:1 size:[40 20]]]",
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

// gestureProgram logs the pointer events of a window as short lines.
const gestureProgram = `
let w = window.open(40, 30, "app", {"headless": true, "scale": 2, "fps": 240})
let log = []
window.automation(w)
window.run(w, function(events) {
  for (let i = 0; i < len(events); i += 1) {
    let e = events[i]
    if (e.type == "move" || e.type == "down" || e.type == "up") {
      if (e.type == "move") {
        push(log, [e.type, e.pos, e.pen.pressure, e.pen.source, len(e.samples)])
      } else {
        push(log, [e.type, e.pos, e.button, e.pen.pressure, e.mods.shift])
      }
    } else {
      push(log, [e.type])
    }
  }
  window.clear(w, #336699)
})
log`

func TestGestures(t *testing.T) {
	want := `[` +
		// Press: a hover, then the button, with the pressure of a tablet.
		`["move", (2, 2), 0.0, "tablet", 1], ["down", (2, 2), "right", 0.5, true], ` +
		// Move while held: the pressure goes on, three samples in each event.
		`["move", (6, 2), 0.5, "tablet", 3], ["move", (10, 2), 0.5, "tablet", 3], ` +
		// Release elsewhere: a move there first.
		`["move", (10, 6), 0.5, "tablet", 1], ["up", (10, 6), "right", 0.5, true], ` +
		// Move with nothing held is a hover; --full is in screen pixels.
		`["move", (5, 5), 1.0, "mouse", 1], ` +
		// A closed path of three points with a ramp, spread along its length:
		// a hover, the press, 2 moves per line, the release at the start.
		`["move", (0, 0), 0.0, "tablet", 1], ["down", (0, 0), "left", 0.0, false], ` +
		`["move", (2, 0), 0.14644660940672624, "tablet", 1], ["move", (4, 0), 0.2928932188134525, "tablet", 1], ` +
		`["move", (4, 2), 0.4393398282201787, "tablet", 1], ["move", (4, 4), 0.585786437626905, "tablet", 1], ` +
		`["move", (2, 2), 0.7928932188134525, "tablet", 1], ["move", (0, 0), 1.0, "tablet", 1], ["up", (0, 0), "left", 1.0, false], ` +
		// Press and Release with nothing between, then a Click: both hover first.
		`["move", (1, 1), 1.0, "mouse", 1], ["down", (1, 1), "left", 1.0, false], ["up", (1, 1), "left", 1.0, false], ` +
		`["move", (3, 3), 1.0, "mouse", 1], ["down", (3, 3), "left", 1.0, false], ["up", (3, 3), "left", 1.0, false], ` +
		`["close"]]`
	session := func(dir string) string {
		var problems []string
		send := func(args ...string) map[string]any {
			reply, err := drive(t, args...)
			if err != nil {
				problems = append(problems, sprintf("%v: %v %v", args, err, reply))
			}
			return reply
		}
		fails := func(sub string, args ...string) {
			reply, err := drive(t, args...)
			if err == nil || !strings.Contains(sprintf("%v", reply["error"]), sub) {
				problems = append(problems, sprintf("%v gave %v, want an error with %q", args, reply, sub))
			}
		}
		send("Press", "2", "2", "shift", "--button", "right", "--pressure", "0.5")
		send("Move", "10", "2", "--steps", "2", "--samples", "3")
		send("Release", "10", "6")
		send("Move", "10", "10", "--full")
		send("Path", "0", "0", "4", "0", "4", "4", "--steps", "2", "--close", "--pressure", "0,1")
		fails("no button is down", "Release", "1", "1")
		send("Press", "1", "1")
		fails("already down", "Press", "1", "1")
		fails("already down", "Path", "0", "0", "1", "1")
		send("Release", "1", "1")
		fails("usage: Press", "Press", "1")
		fails("bad pressure", "Press", "1", "1", "--pressure", "2")
		fails("one pressure", "Press", "1", "1", "--pressure", "0.1,0.2")
		fails("--button must be", "Press", "1", "1", "--button", "back")
		fails("--steps must be", "Move", "1", "1", "--steps", "0")
		fails("--samples must be", "Move", "1", "1", "--samples", "x")
		fails("pairs of numbers", "Path", "1", "1", "2")
		fails("pairs of numbers", "Path", "1", "1")
		fails("more than", "Path", "0", "0", "1", "1", "2", "2", "3", "3", "4", "4", "5", "5", "6", "6", "7", "7", "8", "8", "9", "9", "10", "10", "11", "11", "--steps", "10000")
		send("Click", "3", "3")
		send("Quit")
		return strings.Join(problems, "\n")
	}
	for i, engine := range engines {
		t.Run(engine.name, func(t *testing.T) {
			got, problem := runDriven(t, i, gestureProgram, session)
			if problem != "" {
				t.Fatal(problem)
			}
			if got != want {
				t.Errorf("got\n%s\nwant\n%s", got, want)
			}
		})
	}
}

// dropProgram logs the drop events of a window.
const dropProgram = `
let w = window.open(40, 30, "app", {"headless": true, "scale": 2, "fps": 240})
let log = []
window.surface(w, "canvas", [20, 0, 20, 30], 1)
window.automation(w)
window.inject(w, {"type": "drop", "pos": (1, 2), "paths": ["/a/b.png"]})
window.run(w, function(events) {
  for (let i = 0; i < len(events); i += 1) {
    let e = events[i]
    if (e.type == "drop") {
      let on = "-"
      if (hash.has(e, "surface")) { on = e.surface }
      push(log, [e.pos, e.paths, on])
    } elif (e.type == "close") {
      push(log, "close")
    }
  }
  window.clear(w, #336699)
})
log`

func TestDrop(t *testing.T) {
	want := `[[(1, 2), ["/a/b.png"], "-"], [(25, 3), ["/x/y.png", "/z"], "canvas"], "close"]`
	session := func(dir string) string {
		reply, err := drive(t, "Drop", "25", "3", "/x/y.png", "/z")
		if err != nil {
			return sprintf("Drop: %v %v", err, reply)
		}
		if reply, err = drive(t, "Drop", "1", "1"); err == nil {
			return sprintf("Drop with no path gave %v", reply)
		}
		drive(t, "Quit")
		return ""
	}
	for i, engine := range engines {
		t.Run(engine.name, func(t *testing.T) {
			got, problem := runDriven(t, i, dropProgram, session)
			if problem != "" {
				t.Fatal(problem)
			}
			if got != want {
				t.Errorf("got\n%s\nwant\n%s", got, want)
			}
		})
	}
}

// scrollProgram logs the scroll events of a window.
const scrollProgram = `
let w = window.open(40, 30, "app", {"headless": true, "scale": 2, "fps": 240})
let log = []
window.automation(w)
window.inject(w, {"type": "scroll", "pos": (1, 2), "dy": 2, "dx": -1})
window.run(w, function(events) {
  for (let i = 0; i < len(events); i += 1) {
    let e = events[i]
    if (e.type == "scroll") {
      push(log, [e.dy, e.dx])
    } elif (e.type == "close") {
      push(log, "close")
    }
  }
  window.clear(w, #336699)
})
log`

func TestScroll(t *testing.T) {
	want := `[[2.0, -1.0], [3.0, 0.0], [0.0, 4.5], [-1.0, 2.0], "close"]`
	session := func(dir string) string {
		for _, args := range [][]string{
			{"3"}, {"0", "--dx", "4.5"}, {"-1", "5", "5", "--dx=2"},
		} {
			if reply, err := drive(t, append([]string{"Scroll"}, args...)...); err != nil {
				return sprintf("Scroll %v: %v %v", args, err, reply)
			}
		}
		if reply, err := drive(t, "Scroll", "1", "--dx", "x"); err == nil {
			return sprintf("a bad --dx gave %v", reply)
		}
		drive(t, "Quit")
		return ""
	}
	for i, engine := range engines {
		t.Run(engine.name, func(t *testing.T) {
			got, problem := runDriven(t, i, scrollProgram, session)
			if problem != "" {
				t.Fatal(problem)
			}
			if got != want {
				t.Errorf("got\n%s\nwant\n%s", got, want)
			}
		})
	}
}

// pinchProgram logs the pinch and keyup events of a window.
const pinchProgram = `
let w = window.open(40, 30, "app", {"headless": true, "scale": 2, "fps": 240})
let log = []
window.automation(w)
window.inject(w, {"type": "pinch", "pos": (1, 2), "scale": 1.5})
window.inject(w, {"type": "keyup", "key": "Space"})
window.run(w, function(events) {
  for (let i = 0; i < len(events); i += 1) {
    let e = events[i]
    if (e.type == "pinch") {
      push(log, ["pinch", e.scale, e.pos, e.mods.cmd])
    } elif (e.type == "keyup") {
      push(log, ["keyup", e.key, e.mods.shift])
    } elif (e.type == "close") {
      push(log, "close")
    }
  }
  window.clear(w, #336699)
})
log`

func TestPinchAndKeyUp(t *testing.T) {
	want := `[["pinch", 1.5, (1, 2), false], ["keyup", "Space", false], ["pinch", 0.5, (0, 0), false], ["pinch", 2.0, (5, 6), true], ` +
		`["pinch", 1.25, (10, 10), false], ["keyup", "A", true], "close"]`
	session := func(dir string) string {
		defer drive(t, "Quit")
		for _, args := range [][]string{
			{"Pinch", "0.5"}, {"Pinch", "2", "5", "6", "cmd"}, {"Pinch", "1.25", "20", "20", "--full"}, {"KeyUp", "A", "shift"},
		} {
			if reply, err := drive(t, args...); err != nil {
				return sprintf("%v: %v %v", args, err, reply)
			}
		}
		for _, args := range [][]string{{"Pinch"}, {"Pinch", "x"}, {"Pinch", "0"}, {"Pinch", "1", "2", "x"}, {"KeyUp"}} {
			if reply, err := drive(t, args...); err == nil {
				return sprintf("%v gave %v", args, reply)
			}
		}
		return ""
	}
	for i, engine := range engines {
		t.Run(engine.name, func(t *testing.T) {
			got, problem := runDriven(t, i, pinchProgram, session)
			if problem != "" {
				t.Fatal(problem)
			}
			if got != want {
				t.Errorf("got\n%s\nwant\n%s", got, want)
			}
		})
	}
}
