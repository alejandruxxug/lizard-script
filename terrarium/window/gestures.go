package window

import (
	"fmt"
	"lizardscript/terrarium/pen"
	"math"
	"strconv"
	"strings"
)

// The gesture commands: Press, Move, Release and Path make the events of a
// hand one part at a time (or all at once), so a driver can look at the
// window in the middle of a stroke. Every one sends the events a real
// pointer would: a hover move before a press, moves with samples, a release
// at the place the pointer is.

// maxMoves is the most move events one command makes.
const maxMoves = 100000

// held is the button that Press holds until Release.
type held struct {
	button string
	mods   Mods
	pen    Pen
}

// parsePressure reads --pressure: numbers from 0 to 1 divided by commas.
func parsePressure(s string) ([]float64, error) {
	var out []float64
	if s == "" {
		return out, nil
	}
	for _, part := range strings.Split(s, ",") {
		v, err := strconv.ParseFloat(strings.TrimSpace(part), 64)
		if err != nil || math.IsNaN(v) || v < 0 || v > 1 {
			return nil, fmt.Errorf("bad pressure %q: want numbers from 0 to 1, like 0.2,1.0", part)
		}
		out = append(out, v)
	}
	return out, nil
}

// rampPen is the pen at t (0 to 1) of a gesture: base when there is no
// ramp, else a tablet whose pressure goes through the values of the ramp.
func rampPen(ramp []float64, base Pen) func(t float64) Pen {
	return func(t float64) Pen {
		if len(ramp) == 0 {
			return base
		}
		p := ramp[0]
		if len(ramp) > 1 {
			segment := math.Max(0, math.Min(1, t)) * float64(len(ramp)-1)
			i := min(int(segment), len(ramp)-2)
			p = ramp[i] + (ramp[i+1]-ramp[i])*(segment-float64(i))
		}
		return Pen{Pressure: p, Raw: p, Source: "tablet"}
	}
}

// countFlag reads a whole number flag from 1 to limit, or otherwise.
func countFlag(flags map[string]string, name string, otherwise, limit int) (int, error) {
	v, ok := flags[name]
	if !ok {
		return otherwise, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 1 || n > limit {
		return 0, fmt.Errorf("--%s must be a whole number from 1 to %d, got %q", name, limit, v)
	}
	return n, nil
}

// buttonFlag reads --button.
func buttonFlag(flags map[string]string) (string, error) {
	switch b := flags["button"]; b {
	case "":
		return "left", nil
	case "left", "right", "middle":
		return b, nil
	default:
		return "", fmt.Errorf("--button must be left, right or middle, got %q", b)
	}
}

// injectChecked is injectAfter for events that can fail once the window's
// state is known (a Release with nothing held).
func (a *automation) injectChecked(make func() ([]Event, error)) (any, error) {
	return a.do(func() func() (any, error) {
		events, err := make()
		if err != nil {
			return func() (any, error) { return nil, err }
		}
		a.win.queued = append(a.win.queued, events...)
		return a.state
	})
}

// moveTo is the moves from the pointer's place to (x, y), in screen pixels:
// steps events, each with samples samples when that is more than 1 (every
// point the system would have sent since the last frame). t0 to t1 is the
// part of the gesture it covers, for penAt.
func (a *automation) moveTo(x, y float64, steps, samples int, mods Mods, penAt func(float64) Pen, t0, t1 float64) []Event {
	w := a.win
	x0, y0 := x, y
	if a.known {
		x0, y0 = a.pos[0], a.pos[1]
	}
	var events []Event
	for i := 1; i <= steps; i++ {
		prev, f := float64(i-1)/float64(steps), float64(i)/float64(steps)
		at := func(f float64) (float64, float64) { return x0 + (x-x0)*f, y0 + (y-y0)*f }
		ex, ey := at(f)
		e := w.at(Event{Type: "move", Mods: mods, Pen: penAt(t0 + (t1-t0)*f), T: pen.Now()}, ex, ey)
		if samples > 1 {
			for j := 1; j <= samples; j++ {
				g := prev + (f-prev)*float64(j)/float64(samples)
				sx, sy := at(g)
				p := penAt(t0 + (t1-t0)*g)
				s := w.at(Event{}, sx, sy)
				e.Samples = append(e.Samples, Sample{X: s.X, Y: s.Y, Spot: s.Spot, Pressure: p.Pressure, Raw: p.Raw, T: pen.Now()})
			}
		}
		events = append(events, e)
	}
	a.pos, a.known = [2]float64{x, y}, true
	return events
}

// hover is one plain move to (x, y): the pointer arrives before it presses.
func (a *automation) hover(x, y float64, mods Mods, p Pen) []Event {
	return a.moveTo(x, y, 1, 1, mods, func(float64) Pen { return p }, 0, 1)
}

// press is Press: a hover move, then the button goes down. The button stays
// down until Release.
func (a *automation) press(args []string) (any, error) {
	words, flags := split(args)
	usage := commands["Press"].usage
	f, err := numbers(words, 2, usage)
	if err != nil {
		return nil, err
	}
	button, err := buttonFlag(flags)
	if err != nil {
		return nil, err
	}
	ramp, err := parsePressure(flags["pressure"])
	if err != nil {
		return nil, err
	}
	if len(ramp) > 1 {
		return nil, fmt.Errorf("Press takes one pressure, got %d (usage: %s)", len(ramp), usage)
	}
	mods := modsOf(words[2:], flags)
	return a.injectChecked(func() ([]Event, error) {
		if a.held != nil {
			return nil, fmt.Errorf("the %s button is already down (Release it first)", a.held.button)
		}
		p, hoverPen := mouse, mouse
		if len(ramp) == 1 {
			p = Pen{Pressure: ramp[0], Raw: ramp[0], Source: "tablet"}
			hoverPen = Pen{Source: "tablet"}
		}
		sx, sy := a.screenPoint(f[0], f[1], flags)
		events := a.hover(sx, sy, mods, hoverPen)
		events = append(events, a.win.at(Event{Type: "down", Button: button, Mods: mods, Pen: p, T: pen.Now()}, sx, sy))
		a.held = &held{button: button, mods: mods, pen: p}
		return events, nil
	})
}

// move is Move: the pointer goes to a point, in steps, hovering or, after
// Press, dragging.
func (a *automation) move(args []string) (any, error) {
	words, flags := split(args)
	usage := commands["Move"].usage
	f, err := numbers(words, 2, usage)
	if err != nil {
		return nil, err
	}
	ramp, err := parsePressure(flags["pressure"])
	if err != nil {
		return nil, err
	}
	steps, err := countFlag(flags, "steps", 1, maxMoves)
	if err != nil {
		return nil, err
	}
	samples, err := countFlag(flags, "samples", 1, 1000)
	if err != nil {
		return nil, err
	}
	mods := modsOf(words[2:], flags)
	return a.injectChecked(func() ([]Event, error) {
		base := mouse
		if a.held != nil {
			base, mods = a.held.pen, a.held.mods
		}
		penAt := rampPen(ramp, base)
		sx, sy := a.screenPoint(f[0], f[1], flags)
		events := a.moveTo(sx, sy, steps, samples, mods, penAt, 0, 1)
		if a.held != nil {
			a.held.pen = penAt(1)
		}
		return events, nil
	})
}

// release is Release: the button goes up at a point, after a move there
// when the pointer is elsewhere.
func (a *automation) release(args []string) (any, error) {
	words, flags := split(args)
	f, err := numbers(words, 2, commands["Release"].usage)
	if err != nil {
		return nil, err
	}
	return a.injectChecked(func() ([]Event, error) {
		h := a.held
		if h == nil {
			return nil, fmt.Errorf("no button is down (Press one first)")
		}
		sx, sy := a.screenPoint(f[0], f[1], flags)
		var events []Event
		if !a.known || a.pos != [2]float64{sx, sy} {
			events = a.moveTo(sx, sy, 1, 1, h.mods, func(float64) Pen { return h.pen }, 0, 1)
		}
		events = append(events, a.win.at(Event{Type: "up", Button: h.button, Mods: h.mods, Pen: h.pen, T: pen.Now()}, sx, sy))
		a.held = nil
		return events, nil
	})
}

// path is Path: a hover, a press at the first point, moves along the lines
// to the others (and back to the first with --close) and a release at the
// end. The pressure ramp is spread along the length of the path.
func (a *automation) path(args []string) (any, error) {
	words, flags := split(args)
	usage := commands["Path"].usage
	n := 0
	for n < len(words) {
		if _, err := strconv.ParseFloat(words[n], 64); err != nil {
			break
		}
		n++
	}
	if n < 4 || n%2 != 0 {
		return nil, fmt.Errorf("a path wants pairs of numbers, at least two points (usage: %s)", usage)
	}
	f, err := numbers(words, n, usage)
	if err != nil {
		return nil, err
	}
	button, err := buttonFlag(flags)
	if err != nil {
		return nil, err
	}
	ramp, err := parsePressure(flags["pressure"])
	if err != nil {
		return nil, err
	}
	steps, err := countFlag(flags, "steps", 10, maxMoves)
	if err != nil {
		return nil, err
	}
	samples, err := countFlag(flags, "samples", 1, 1000)
	if err != nil {
		return nil, err
	}
	_, closed := flags["close"]
	points := make([][2]float64, 0, n/2+1)
	for i := 0; i < n; i += 2 {
		points = append(points, [2]float64{f[i], f[i+1]})
	}
	if closed {
		points = append(points, points[0])
	}
	if (len(points)-1)*steps > maxMoves {
		return nil, fmt.Errorf("the path would make more than %d moves (lower --steps)", maxMoves)
	}
	mods := modsOf(words[n:], flags)
	return a.injectChecked(func() ([]Event, error) {
		if a.held != nil {
			return nil, fmt.Errorf("the %s button is already down (Release it first)", a.held.button)
		}
		// Where each point is along the path, 0 to 1, by length.
		at := make([]float64, len(points))
		for i := 1; i < len(points); i++ {
			at[i] = at[i-1] + math.Hypot(points[i][0]-points[i-1][0], points[i][1]-points[i-1][1])
		}
		for i := range at {
			if at[len(at)-1] > 0 {
				at[i] /= at[len(at)-1]
			} else {
				at[i] = float64(i) / float64(len(at)-1)
			}
		}
		base, hoverPen := mouse, mouse
		if len(ramp) > 0 {
			hoverPen = Pen{Source: "tablet"}
		}
		penAt := rampPen(ramp, base)
		sx, sy := a.screenPoint(points[0][0], points[0][1], flags)
		events := a.hover(sx, sy, mods, hoverPen)
		events = append(events, a.win.at(Event{Type: "down", Button: button, Mods: mods, Pen: penAt(0), T: pen.Now()}, sx, sy))
		for i := 1; i < len(points); i++ {
			sx, sy = a.screenPoint(points[i][0], points[i][1], flags)
			events = append(events, a.moveTo(sx, sy, steps, samples, mods, penAt, at[i-1], at[i])...)
		}
		return append(events, a.win.at(Event{Type: "up", Button: button, Mods: mods, Pen: penAt(1), T: pen.Now()}, sx, sy)), nil
	})
}
