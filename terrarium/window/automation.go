package window

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"lizardscript/object"
	"lizardscript/terrarium"
	"lizardscript/terrarium/internal/arg"
	"lizardscript/terrarium/pen"
	"math"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Automation: a window can let another program drive it, the way a person
// would, to test it or to let an agent use it. It listens on 127.0.0.1 on
// a free port and writes the port and a secret token to its own drive file
// (DriveFile), readable only by the user; `lizardscript drive` reads that
// file and sends one command (Drive). Input commands go through the same
// path as real input, and every reply comes after the frame that handled
// the command, so it sees what the program did.

// AutomationEnv turns automation on for every window a program opens, when
// it is "1".
const AutomationEnv = "LIZARD_AUTOMATION"

// TokenHeader carries the token of the drive file.
const TokenHeader = "X-Lizard-Token"

// automation is the server of one window.
type automation struct {
	win   *Window
	token string
	srv   *http.Server
	file  string
	jobs  chan job
	mu    sync.Mutex
	done  chan struct{} // closed when the window stops; under mu
	shots int

	// The program's hooks (window.automation): run on the main goroutine.
	dump, stats object.Object // functions of no arguments, or nil
	commands    *object.Hash  // name -> function(args), or nil

	waiters []*waiter
	quit    bool

	// Where the driver's pointer is (screen pixels, once it has moved) and
	// the button Press holds; used on the main goroutine.
	pos   [2]float64
	known bool
	held  *held
}

// job is work for the main goroutine. run returns the function that makes
// the reply; it is called after the frame's tick, so the reply sees it.
type job struct {
	run   func() func() (any, error)
	reply chan result
}

type result struct {
	value any
	err   error
}

// waiter is a Wait command: it is answered at the first idle frame.
type waiter struct {
	reply  chan result
	frames int
}

// active is the automation that runs now: one window of a program at a
// time can be driven, since the drive file is the program's.
var active *automation

func newAutomation() *automation {
	return &automation{jobs: make(chan job, 64)}
}

// start listens and writes the drive file. Files of programs that no
// longer run are removed first.
func (a *automation) start(w *Window) error {
	if active != nil {
		return fmt.Errorf("another window of this program is automated")
	}
	secret := make([]byte, 16)
	if _, err := rand.Read(secret); err != nil {
		return err
	}
	a.win, a.token, a.quit = w, hex.EncodeToString(secret), false
	a.known, a.held = false, nil
	a.mu.Lock()
	a.done = make(chan struct{})
	a.mu.Unlock()
	dir, err := driveDir()
	if err != nil {
		return err
	}
	cleanStale(dir)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/cmd", a.serve)
	a.srv = &http.Server{Handler: mux}
	go a.srv.Serve(ln)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		a.srv.Close()
		return err
	}
	a.file = DriveFile(dir, os.Getpid())
	info, _ := json.Marshal(driveInfo{Port: ln.Addr().(*net.TCPAddr).Port, Token: a.token, PID: os.Getpid(), Title: w.title})
	if err := os.WriteFile(a.file, info, 0o600); err != nil {
		a.srv.Close()
		return err
	}
	active = a
	return nil
}

// stop closes the server, once the replies being written are sent,
// removes the drive file and answers what still waits.
func (a *automation) stop() {
	a.mu.Lock()
	if a.done != nil {
		close(a.done)
		a.done = nil
	}
	a.mu.Unlock()
	if a.srv != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		a.srv.Shutdown(ctx)
		cancel()
	}
	if a.file != "" {
		os.Remove(a.file)
	}
	for _, w := range a.waiters {
		w.reply <- result{err: errClosed}
	}
	a.waiters, a.srv, a.file = nil, nil, ""
	if active == a {
		active = nil
	}
}

// runJobs runs the commands that came since the last frame, and gives the
// functions that make their replies.
func (a *automation) runJobs() []func() {
	if a == nil {
		return nil
	}
	var replies []func()
	for {
		select {
		case j := <-a.jobs:
			thunk := j.run()
			replies = append(replies, func() {
				v, err := thunk()
				j.reply <- result{v, err}
			})
		default:
			return replies
		}
	}
}

// reply answers the commands of this frame, and the waiters once a frame
// had nothing to do.
func (a *automation) reply(replies []func(), busy bool) {
	if a == nil {
		return
	}
	for _, r := range replies {
		r()
	}
	keep := a.waiters[:0]
	for _, w := range a.waiters {
		w.frames++
		if (!busy && w.frames > 1) || w.frames > 90 {
			w.reply <- result{value: map[string]any{"idle": !busy, "frames": w.frames}}
		} else {
			keep = append(keep, w)
		}
	}
	a.waiters = keep
}

// quitting reports whether a Quit command closed the window.
func (a *automation) quitting() bool { return a != nil && a.quit }

// do runs f on the main goroutine, in the next frame, and waits for the
// reply it makes.
func (a *automation) do(f func() func() (any, error)) (any, error) {
	j := job{run: f, reply: make(chan result, 1)}
	a.mu.Lock()
	done := a.done
	a.mu.Unlock()
	if done == nil {
		return nil, errClosed
	}
	select {
	case a.jobs <- j:
	case <-done:
		return nil, errClosed
	case <-time.After(30 * time.Second):
		return nil, fmt.Errorf("the window is not taking commands")
	}
	select {
	case r := <-j.reply:
		return r.value, r.err
	case <-done:
		return nil, errClosed
	case <-time.After(30 * time.Second):
		return nil, fmt.Errorf("timed out waiting for the window")
	}
}

var errClosed = errors.New("the window closed")

type request struct {
	Cmd  string   `json:"cmd"`
	Args []string `json:"args"`
}

func (a *automation) serve(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get(TokenHeader) != a.token {
		http.Error(w, `{"error":"missing or wrong token"}`, http.StatusForbidden)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"POST only"}`, http.StatusMethodNotAllowed)
		return
	}
	body, _ := io.ReadAll(io.LimitReader(r.Body, 8<<20))
	var req request
	if err := json.Unmarshal(body, &req); err != nil {
		send(w, nil, fmt.Errorf("bad request: %v", err))
		return
	}
	command, ok := commands[req.Cmd]
	if !ok {
		send(w, nil, fmt.Errorf("unknown command %q (try List)", req.Cmd))
		return
	}
	out, err := command.run(a, req.Args)
	send(w, out, err)
}

func send(w http.ResponseWriter, out any, err error) {
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{"error": err.Error()})
		return
	}
	json.NewEncoder(w).Encode(out)
}

// command is one thing a driver can ask, with how it is used.
type command struct {
	usage string
	run   func(a *automation, args []string) (any, error)
}

var commands map[string]command

func init() {
	commands = map[string]command{
		"List":        {"List", (*automation).list},
		"Dump":        {"Dump", (*automation).dumpCmd},
		"Stats":       {"Stats", (*automation).statsCmd},
		"Screenshot":  {"Screenshot [path] [--base64]", (*automation).screenshot},
		"Click":       {"Click x y [shift|alt|cmd|ctrl] [--button right] [--full]", clicks(1)},
		"Press":       {"Press x y [mods] [--button right] [--pressure p] [--full]", (*automation).press},
		"Move":        {"Move x y [mods] [--pressure a,b,...] [--steps n] [--samples k] [--full]", (*automation).move},
		"Release":     {"Release x y [--full]", (*automation).release},
		"Path":        {"Path x1 y1 x2 y2 ... [mods] [--pressure a,b,...] [--steps n] [--samples k] [--close] [--button right] [--full]", (*automation).path},
		"DoubleClick": {"DoubleClick x y [mods] [--button right] [--full]", clicks(2)},
		"Drag":        {"Drag x1 y1 x2 y2 [steps] [mods] [--pressure 0.2,1.0] [--full]", (*automation).drag},
		"Scroll":      {"Scroll dy [x y] [--full]", (*automation).scroll},
		"Key":         {"Key name [shift|alt|cmd|ctrl]", (*automation).key},
		"Type":        {"Type text", (*automation).typeText},
		"Resize":      {"Resize w h", (*automation).resize},
		"Focus":       {"Focus true|false", (*automation).focus},
		"Leave":       {"Leave", presenceCmd("leave")},
		"Enter":       {"Enter", presenceCmd("enter")},
		"Wait":        {"Wait", (*automation).wait},
		"Quit":        {"Quit", (*automation).quitCmd},
		"Call":        {"Call Name args...", (*automation).call},
	}
}

// ---- reading the arguments ----

// split separates the --flags from the words. A flag takes the next word
// as its value unless it is one that stands alone.
func split(args []string) (words []string, flags map[string]string) {
	flags = map[string]string{}
	alone := map[string]bool{"full": true, "base64": true, "close": true, "shift": true, "alt": true, "cmd": true, "ctrl": true}
	for i := 0; i < len(args); i++ {
		a := args[i]
		if !strings.HasPrefix(a, "--") {
			words = append(words, a)
			continue
		}
		name, value, has := strings.Cut(a[2:], "=")
		if !has && !alone[name] && i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
			i++
			value = args[i]
		}
		flags[name] = value
	}
	return words, flags
}

// numbers reads the first n words as numbers.
func numbers(words []string, n int, usage string) ([]float64, error) {
	if len(words) < n {
		return nil, fmt.Errorf("usage: %s", usage)
	}
	out := make([]float64, n)
	for i := range out {
		f, err := strconv.ParseFloat(words[i], 64)
		if err != nil || math.IsInf(f, 0) || math.IsNaN(f) {
			return nil, fmt.Errorf("%q is not a number (usage: %s)", words[i], usage)
		}
		out[i] = f
	}
	return out, nil
}

// modsOf reads shift, alt, cmd and ctrl from words (shift+cmd counts) and
// flags.
func modsOf(words []string, flags map[string]string) Mods {
	var m Mods
	set := func(word string) {
		switch strings.ToLower(word) {
		case "shift":
			m.Shift = true
		case "alt", "option":
			m.Alt = true
		case "cmd", "meta", "command":
			m.Cmd = true
		case "ctrl", "control":
			m.Ctrl = true
		}
	}
	for _, word := range words {
		for _, part := range strings.Split(word, "+") {
			set(part)
		}
	}
	for name := range flags {
		set(name)
	}
	return m
}

// screenPoint is the point (x, y) of a command in screen pixels: x and y
// are buffer pixels, what events call pos, or screen pixels with --full.
func (a *automation) screenPoint(x, y float64, flags map[string]string) (float64, float64) {
	if _, full := flags["full"]; full {
		return x, y
	}
	k := float64(a.win.scale)
	return x * k, y * k
}

// injectAfter queues the events made by make in the next frame, and
// replies after it.
func (a *automation) injectAfter(make func() []Event) (any, error) {
	return a.do(func() func() (any, error) {
		a.win.queued = append(a.win.queued, make()...)
		return a.state
	})
}

// state is the reply of an input command.
func (a *automation) state() (any, error) {
	return map[string]any{"ticks": a.win.stats.ticks, "frames": a.win.stats.frames}, nil
}

// ---- commands ----

func (a *automation) list(args []string) (any, error) {
	names := make([]string, 0, len(commands))
	for name := range commands {
		names = append(names, name)
	}
	sort.Strings(names)
	return a.do(func() func() (any, error) {
		app := []string{}
		if a.commands != nil {
			a.commands.Each(func(p object.HashPair) { app = append(app, label(p.Key)) })
		}
		sort.Strings(app)
		return func() (any, error) { return map[string]any{"commands": names, "app": app}, nil }
	})
}

func (a *automation) dumpCmd(args []string) (any, error) {
	return a.do(func() func() (any, error) {
		return func() (any, error) {
			w := a.win
			fw, fh := w.frameSize()
			sw, sh := w.screenSize()
			surfaces := []any{}
			for _, s := range w.screen.surfaces {
				surfaces = append(surfaces, s.describe())
			}
			out := map[string]any{
				"window": map[string]any{
					"title": w.title, "size": []int{w.buf.W, w.buf.H}, "scale": w.scale,
					"frame": []int{fw, fh}, "screen": []int{sw, sh},
					"frameless": w.frameless, "headless": w.headless, "on_demand": w.onDemand,
				},
				"surfaces": surfaces,
				"stats":    a.statsMap(),
			}
			if a.dump != nil {
				v, err := a.callHook(a.dump)
				if err != nil {
					out["app_error"] = err.Error()
				} else {
					out["app"] = v
				}
			}
			return out, nil
		}
	})
}

func (a *automation) statsMap() map[string]any {
	s := a.win.stats
	out := map[string]any{"ticks": s.ticks, "frames": s.frames, "events": s.events, "tick_ms": s.tickMS, "compose_ms": s.composeMS}
	if a.stats != nil {
		if v, err := a.callHook(a.stats); err == nil {
			out["app"] = v
		}
	}
	return out
}

func (a *automation) statsCmd(args []string) (any, error) {
	return a.do(func() func() (any, error) {
		return func() (any, error) { return a.statsMap(), nil }
	})
}

func (a *automation) screenshot(args []string) (any, error) {
	words, flags := split(args)
	_, inline := flags["base64"]
	path := ""
	if len(words) > 0 {
		path = words[0]
	}
	return a.do(func() func() (any, error) {
		return func() (any, error) {
			shot := a.win.shot()
			var data bytes.Buffer
			if err := shot.EncodePNG(&data); err != nil {
				return nil, err
			}
			out := map[string]any{"width": shot.W, "height": shot.H}
			if inline && path == "" {
				out["png"] = base64.StdEncoding.EncodeToString(data.Bytes())
				return out, nil
			}
			if path == "" {
				a.shots++
				path = filepath.Join(os.TempDir(), fmt.Sprintf("lizard-shot-%d-%d.png", os.Getpid(), a.shots))
			}
			if abs, err := filepath.Abs(path); err == nil {
				path = abs
			}
			if err := terrarium.WriteFileAtomic(path, data.Bytes()); err != nil {
				return nil, err
			}
			out["path"] = path
			return out, nil
		}
	})
}

// clicks is Click (n = 1) and DoubleClick (n = 2). The pointer moves to the
// point first, as a hand arrives before it presses.
func clicks(n int) func(a *automation, args []string) (any, error) {
	return func(a *automation, args []string) (any, error) {
		words, flags := split(args)
		f, err := numbers(words, 2, commands["Click"].usage)
		if err != nil {
			return nil, err
		}
		button := "left"
		if b := flags["button"]; b != "" {
			button = b
		}
		mods := modsOf(words[2:], flags)
		return a.injectAfter(func() []Event {
			sx, sy := a.screenPoint(f[0], f[1], flags)
			events := a.hover(sx, sy, mods, mouse)
			for i := 0; i < n; i++ {
				for _, kind := range []string{"down", "up"} {
					events = append(events, a.win.at(Event{Type: kind, Button: button, Mods: mods, Pen: mouse, T: pen.Now()}, sx, sy))
				}
			}
			return events
		})
	}
}

// drag is Drag: a press at the first point, steps moves to the second and
// a release there. --pressure a,b,... ramps the pen's pressure along the
// way, from a tablet; without it the drag is a mouse's.
func (a *automation) drag(args []string) (any, error) {
	words, flags := split(args)
	usage := commands["Drag"].usage
	f, err := numbers(words, 4, usage)
	if err != nil {
		return nil, err
	}
	steps, rest := 10, words[4:]
	if len(rest) > 0 {
		if n, err := strconv.Atoi(rest[0]); err == nil {
			if n < 1 || n > 10000 {
				return nil, fmt.Errorf("the steps must be from 1 to 10000, got %d", n)
			}
			steps, rest = n, rest[1:]
		}
	}
	ramp, err := parsePressure(flags["pressure"])
	if err != nil {
		return nil, err
	}
	mods := modsOf(rest, flags)
	penAt := rampPen(ramp, mouse)
	return a.injectAfter(func() []Event {
		w := a.win
		point := func(t float64) (float64, float64) {
			return a.screenPoint(f[0]+(f[2]-f[0])*t, f[1]+(f[3]-f[1])*t, flags)
		}
		x0, y0 := point(0)
		events := []Event{w.at(Event{Type: "down", Button: "left", Mods: mods, Pen: penAt(0), T: pen.Now()}, x0, y0)}
		for i := 1; i <= steps; i++ {
			t := float64(i) / float64(steps)
			x, y := point(t)
			events = append(events, w.at(Event{Type: "move", Mods: mods, Pen: penAt(t), T: pen.Now()}, x, y))
		}
		x1, y1 := point(1)
		a.pos, a.known = [2]float64{x1, y1}, true
		return append(events, w.at(Event{Type: "up", Button: "left", Mods: mods, Pen: penAt(1), T: pen.Now()}, x1, y1))
	})
}

func (a *automation) scroll(args []string) (any, error) {
	words, flags := split(args)
	usage := commands["Scroll"].usage
	f, err := numbers(words, 1, usage)
	if err != nil {
		return nil, err
	}
	at := []float64{0, 0}
	if len(words) >= 3 {
		if at, err = numbers(words[1:], 2, usage); err != nil {
			return nil, err
		}
	}
	return a.injectAfter(func() []Event {
		sx, sy := a.screenPoint(at[0], at[1], flags)
		return []Event{a.win.at(Event{Type: "scroll", DY: f[0], T: pen.Now()}, sx, sy)}
	})
}

func (a *automation) key(args []string) (any, error) {
	words, flags := split(args)
	if len(words) < 1 {
		return nil, fmt.Errorf("usage: %s", commands["Key"].usage)
	}
	mods := modsOf(words[1:], flags)
	return a.injectAfter(func() []Event {
		return []Event{{Type: "key", Key: words[0], Mods: mods, T: pen.Now()}}
	})
}

func (a *automation) typeText(args []string) (any, error) {
	text := strings.Join(args, " ")
	return a.injectAfter(func() []Event {
		var events []Event
		for _, r := range text {
			events = append(events, Event{Type: "type", Text: string(r), T: pen.Now()})
		}
		return events
	})
}

func (a *automation) focus(args []string) (any, error) {
	words, _ := split(args)
	on, err := strconv.ParseBool(strings.Join(words, ""))
	if len(words) != 1 || err != nil {
		return nil, fmt.Errorf("usage: %s", commands["Focus"].usage)
	}
	return a.injectAfter(func() []Event { return []Event{{Type: "focus", Focused: on}} })
}

// presenceCmd is Leave and Enter: the pointer left or came into the window.
func presenceCmd(kind string) func(a *automation, args []string) (any, error) {
	return func(a *automation, args []string) (any, error) {
		return a.injectAfter(func() []Event { return []Event{{Type: kind}} })
	}
}

// resize is Resize w h, in buffer pixels, never below the window's
// smallest size; it replies once the program has handled it.
func (a *automation) resize(args []string) (any, error) {
	words, _ := split(args)
	f, err := numbers(words, 2, commands["Resize"].usage)
	if err != nil {
		return nil, err
	}
	width, height := int(f[0]), int(f[1])
	if width < 1 || height < 1 || width > maxSide || height > maxSide {
		return nil, fmt.Errorf("the size must be from 1 to %d on each side", maxSide)
	}
	if _, err := a.do(func() func() (any, error) {
		a.win.setSize(width, height)
		a.win.chrome = true
		return a.state
	}); err != nil {
		return nil, err
	}
	return a.wait(nil)
}

// wait is Wait: it replies at the first frame with nothing to do.
func (a *automation) wait(args []string) (any, error) {
	ch := make(chan result, 1)
	if _, err := a.do(func() func() (any, error) {
		a.waiters = append(a.waiters, &waiter{reply: ch})
		return func() (any, error) { return nil, nil }
	}); err != nil {
		return nil, err
	}
	select {
	case r := <-ch:
		return r.value, r.err
	case <-time.After(10 * time.Second):
		return nil, fmt.Errorf("timed out waiting for the window to be idle")
	}
}

// quitCmd is Quit: the program gets a close event, as if the user closed
// the window, and the window stops after that frame.
func (a *automation) quitCmd(args []string) (any, error) {
	return a.do(func() func() (any, error) {
		a.win.queued = append(a.win.queued, Event{Type: "close"})
		a.quit = true
		return func() (any, error) { return map[string]any{"quit": true}, nil }
	})
}

// call is Call Name args...: it calls the program's command Name with the
// list of args, numbers as numbers and the rest as strings, and replies
// with what it gives. An error in it is the reply; the window goes on.
func (a *automation) call(args []string) (any, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("usage: %s", commands["Call"].usage)
	}
	name, rest := args[0], args[1:]
	return a.do(func() func() (any, error) {
		var fn object.Object
		if a.commands != nil {
			fn, _ = a.commands.Get(&object.String{Value: name})
		}
		if fn == nil {
			return func() (any, error) { return nil, fmt.Errorf("the program has no command %q (try List)", name) }
		}
		list := &object.Array{}
		for _, word := range rest {
			list.Elements = append(list.Elements, wordValue(word))
		}
		v, err := a.callHook(fn, list)
		return func() (any, error) { return v, err }
	})
}

// callHook calls one of the program's functions and gives its result as
// JSON values.
func (a *automation) callHook(fn object.Object, args ...object.Object) (any, error) {
	out := apply(fn, args...)
	if e, ok := out.(*object.Error); ok {
		return nil, fmt.Errorf("%s", e.Message)
	}
	return jsonOf(out, 0), nil
}

// wordValue is a word of a command as a value: a number when it is one,
// else a string.
func wordValue(s string) object.Object {
	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		return arg.Int64(n)
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil && !math.IsInf(f, 0) && !math.IsNaN(f) {
		return arg.Float(f)
	}
	return arg.Str(s)
}

// label is a hash key as text: a string as it is, anything else as puts
// shows it.
func label(key object.Object) string {
	if s, ok := key.(*object.String); ok {
		return s.Value
	}
	return key.Inspect()
}

// maxDepth is how deep jsonOf goes into arrays and hashes.
const maxDepth = 64

// jsonOf turns a value into one encoding/json writes: a point is a list of
// its coordinates, a color its text, and what JSON has no value for (a
// function, an image) is the text puts shows.
func jsonOf(o object.Object, depth int) any {
	if depth > maxDepth {
		return "..."
	}
	switch v := o.(type) {
	case *object.Integer:
		return v.Value
	case *object.Float:
		return v.Value
	case *object.String:
		return v.Value
	case *object.Boolean:
		return v.Value
	case *object.Null, nil:
		return nil
	case *object.Array:
		out := make([]any, len(v.Elements))
		for i, e := range v.Elements {
			out[i] = jsonOf(e, depth+1)
		}
		return out
	case *object.Point:
		out := make([]any, len(v.Coords))
		for i, c := range v.Coords {
			out[i] = jsonOf(c, depth+1)
		}
		return out
	case *object.Hash:
		out := map[string]any{}
		v.Each(func(p object.HashPair) { out[label(p.Key)] = jsonOf(p.Value, depth+1) })
		return out
	}
	return o.Inspect()
}

// window.automation(w, hooks = true) lets a driver control the window
// (`lizardscript drive`) while it runs. hooks is true, false to turn it
// off, or a hash of the program's own parts:
//
//	"dump":     a function whose value Dump includes, as "app"
//	"stats":    a function whose value Stats includes, as "app"
//	"commands": a hash of name: function(args), for Call name args...
func automationFunc(args ...object.Object) object.Object {
	if err := arg.Count(args, 1, 2); err != nil {
		return err
	}
	w, err := windowArg(args, 0)
	if err != nil {
		return err
	}
	on, hooks := true, (*object.Hash)(nil)
	if len(args) == 2 {
		switch v := args[1].(type) {
		case *object.Boolean:
			on = v.Value
		case *object.Hash:
			hooks = v
		default:
			return arg.Errorf("want true, false or a HASH of hooks, got %s", arg.Name(args[1]))
		}
	}
	if !on {
		if w.auto != nil && w.running {
			w.auto.stop()
		}
		w.auto = nil
		return arg.Null
	}
	a := w.auto
	if a == nil {
		a = newAutomation()
	}
	if hooks != nil {
		if err := a.setHooks(hooks); err != nil {
			return err
		}
	}
	if w.auto == nil && w.running {
		if serr := a.start(w); serr != nil {
			return arg.Errorf("can't start the automation: %v", serr)
		}
	}
	w.auto = a
	return arg.Null
}

// setHooks reads the hooks of window.automation.
func (a *automation) setHooks(h *object.Hash) *object.Error {
	r := reader{h: h}
	var err *object.Error
	h.Each(func(p object.HashPair) {
		if name := label(p.Key); err == nil && name != "dump" && name != "stats" && name != "commands" {
			err = arg.Errorf("unknown hook %q (the hooks are commands, dump, stats)", name)
		}
	})
	if err != nil {
		return err
	}
	function := func(name string) (object.Object, *object.Error) {
		v := r.get(name)
		if v == nil {
			return nil, nil
		}
		if v.Type() != object.FUNCTION_OBJ && v.Type() != object.BUILTIN_OBJ {
			return nil, arg.Errorf("the hook %s must be a FUNCTION, got %s", name, arg.Name(v))
		}
		return v, nil
	}
	if a.dump, err = function("dump"); err != nil {
		return err
	}
	if a.stats, err = function("stats"); err != nil {
		return err
	}
	if v := r.get("commands"); v != nil {
		cmds, ok := v.(*object.Hash)
		if !ok {
			return arg.Errorf("the hook commands must be a HASH of functions, got %s", arg.Name(v))
		}
		a.commands = cmds
	}
	return nil
}
