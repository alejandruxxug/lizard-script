// Package pen reads pen and trackpad pressure for the window module, and is
// the pen module a program sees.
//
// On macOS it listens to the system events the app already receives (Force
// Touch trackpads and tablets); on Windows it reads Windows Ink pointer
// messages; on Linux it reads XInput2 raw events from its own connection to
// the X server (XWayland included). None of them changes what the window
// itself receives. Other systems have no pressure: every pointer is a plain
// mouse.
package pen

import (
	"github.com/alejandruxxug/lizard-script/engine/object"
	"github.com/alejandruxxug/lizard-script/terrarium/internal/arg"
	"os"
	"sync"
	"time"
)

// State is the latest pen or finger reading.
type State struct {
	Pressure float64    // 0 to 1
	Tilt     [2]float64 // tablet tilt, -1 to 1 on each axis
	Rotation float64    // tablet pen rotation in degrees
	Source   string     // "trackpad", "tablet" or "mouse"
	Eraser   bool       // the tablet's eraser end is in use
	OK       bool       // false until something was read
}

var (
	mu    sync.Mutex
	state State
)

// update changes the state under the lock.
func update(f func(s *State)) {
	mu.Lock()
	defer mu.Unlock()
	f(&state)
	state.OK = true
}

// setEraser records which end of the pen is in use.
func setEraser(e bool) {
	mu.Lock()
	state.Eraser = e
	mu.Unlock()
}

// forgetPen hands the reading to a plain pointer: when a mouse or a finger
// moves after the pen, its drag doesn't take the pen's last pressure.
func forgetPen() {
	mu.Lock()
	defer mu.Unlock()
	if state.Source == "tablet" {
		state = State{Source: "mouse", OK: true}
	}
}

// Latest returns the last reading. OK is false when nothing has been read
// yet or the system has no pressure.
func Latest() State {
	mu.Lock()
	defer mu.Unlock()
	return state
}

// Phase says which part of a gesture a sample belongs to.
const (
	PhaseDown = "down"
	PhaseDrag = "drag"
	PhaseUp   = "up"
	PhasePen  = "pen" // a tablet point with no button down (hover)
)

// Sample is one native pointer event. Unlike Latest, nothing is skipped:
// the monitor sees every one, even when the window only polls once a frame.
// X and Y are points in the window's content, from the top left. T is
// seconds since the program started.
type Sample struct {
	T        float64
	X, Y     float64
	Pressure float64
	Tilt     [2]float64
	Rotation float64
	Source   string // "mouse", "trackpad" or "tablet"
	Phase    string
}

// maxQueue bounds the samples kept between two calls of Drain.
const maxQueue = 4096

var (
	queue []Sample
	begin = time.Now()
)

// Now is the clock of Sample.T, in seconds.
func Now() float64 { return time.Since(begin).Seconds() }

// record adds a sample, dropping the oldest ones past maxQueue.
func record(s Sample) {
	mu.Lock()
	defer mu.Unlock()
	queue = append(queue, s)
	if len(queue) > maxQueue {
		queue = append(queue[:0], queue[len(queue)-maxQueue:]...)
	}
}

// Drain returns every sample since the last call, oldest first, and
// forgets them. It is empty when the system has no native events.
func Drain() []Sample {
	mu.Lock()
	defer mu.Unlock()
	out := queue
	queue = nil
	return out
}

// FlipY turns a y measured from the bottom (macOS) into one measured from
// the top of a view that is height points tall.
func FlipY(y, height float64) float64 { return height - y }

// snapshot builds a sample from the state the monitor just updated. x and
// y are measured from the bottom left; height is the content's height.
func snapshot(t, x, y, height float64, phase string) Sample {
	s := Latest()
	if phase == PhaseUp {
		s.Pressure = 0
	}
	src := s.Source
	if src == "" {
		src = "mouse"
	}
	return Sample{T: t, X: x, Y: FlipY(y, height), Pressure: s.Pressure, Tilt: s.Tilt,
		Rotation: s.Rotation, Source: src, Phase: phase}
}

// configOnce makes the user's pen file be read once per program.
var configOnce sync.Once

// Module is the pen module:
//
//	pen.available()              whether this system can report pen pressure at all
//	pen.curve(source)            the response curve of "trackpad", "tablet" or "mouse"
//	pen.curve(source, curve)     changes it
//
// The pressure itself comes in the window's pointer events, as e.pen. The
// first call reads the user's pen file (see LoadUserConfig).
func Module() *object.Hash {
	configOnce.Do(func() { LoadUserConfig(os.Stderr) })
	return arg.Module("pen", map[string]object.BuiltinFunction{
		"curve": curveFunc,
		"available": func(args ...object.Object) object.Object {
			if err := arg.Count(args, 0, 0); err != nil {
				return err
			}
			return arg.Bool(Supported())
		},
	})
}
