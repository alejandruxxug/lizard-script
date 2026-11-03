//go:build darwin

package pen

import (
	"sync"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

// NSEventType values we care about.
const (
	typeLeftMouseDown    = 1
	typeLeftMouseUp      = 2
	typeLeftMouseDragged = 6
	typeTabletPoint      = 23
	typeTabletProximity  = 24
	typePressure         = 34

	subtypeTabletPoint = 1 // NSEventSubtypeTabletPoint
	deviceEraser       = 3 // NSPointingDeviceTypeEraser
)

// nsPoint is an NSPoint (two CGFloats).
type nsPoint struct{ X, Y float64 }

// nsRect is an NSRect.
type nsRect struct{ X, Y, W, H float64 }

// epoch is the system uptime when the app clock started, so event
// timestamps (seconds since boot) become seconds since the app started.
var epoch float64

// sawForce turns true once a Force Touch pressure event arrives: only a Force Touch
// trackpad sends them, so later plain mouse events are trackpad drags.
var sawForce bool

// buttonDown is true between a left mouse down and up.
var buttonDown bool

// monitor is the id returned by addLocalMonitor; kept so it is never freed.
var (
	startOnce sync.Once
	monitor   objc.ID
	handler   objc.Block
)

// Start installs the event monitor. Call it once, on the main thread, after
// the app is running (Ebitengine's first Update is fine). It reports whether
// the monitor was installed.
func Start() bool {
	startOnce.Do(install)
	return monitor != 0
}

// Supported is true: every Mac reports pressure from a Force Touch
// trackpad or a tablet, when there is one.
func Supported() bool { return true }

func install() {
	if _, err := purego.Dlopen("/System/Library/Frameworks/AppKit.framework/AppKit", purego.RTLD_GLOBAL|purego.RTLD_NOW); err != nil {
		return
	}
	var (
		selType     = objc.RegisterName("type")
		selSubtype  = objc.RegisterName("subtype")
		selPressure = objc.RegisterName("pressure")
		selTilt     = objc.RegisterName("tilt")
		selRotation = objc.RegisterName("rotation")
		selEntering = objc.RegisterName("isEnteringProximity")
		selDevice   = objc.RegisterName("pointingDeviceType")
		selTime     = objc.RegisterName("timestamp")
		selLoc      = objc.RegisterName("locationInWindow")
		selWindow   = objc.RegisterName("window")
		selContent  = objc.RegisterName("contentView")
		selFrame    = objc.RegisterName("frame")
	)
	epoch = systemUptime() - Now()
	// sample logs the event as a native sample, if it has a window.
	sample := func(ev objc.ID, phase string) {
		win := objc.Send[objc.ID](ev, selWindow)
		if win == 0 {
			return
		}
		view := objc.Send[objc.ID](win, selContent)
		if view == 0 {
			return
		}
		loc := objc.Send[nsPoint](ev, selLoc)
		frame := objc.Send[nsRect](view, selFrame)
		record(snapshot(objc.Send[float64](ev, selTime)-epoch, loc.X, loc.Y, frame.H, phase))
	}
	// The block must hand the event back so Ebitengine still gets it.
	handler = objc.NewBlock(func(_ objc.Block, ev objc.ID) objc.ID {
		t := objc.Send[uint64](ev, selType)
		switch t {
		case typeTabletProximity:
			if objc.Send[bool](ev, selEntering) {
				setEraser(objc.Send[uint64](ev, selDevice) == deviceEraser)
			} else {
				update(func(s *State) { *s = State{Eraser: s.Eraser, OK: true} })
			}
		case typeTabletPoint:
			readTablet(ev, selPressure, selTilt, selRotation)
			// While a button is held the matching drag event is the sample.
			if !buttonDown {
				sample(ev, PhasePen)
			}
		case typeLeftMouseDown, typeLeftMouseDragged:
			phase := PhaseDrag
			if t == typeLeftMouseDown {
				phase, buttonDown = PhaseDown, true
			}
			if objc.Send[int16](ev, selSubtype) == subtypeTabletPoint {
				readTablet(ev, selPressure, selTilt, selRotation)
				sample(ev, phase)
				break
			}
			p := float64(objc.Send[float32](ev, selPressure))
			update(func(s *State) {
				s.Pressure, s.Tilt, s.Rotation = p, [2]float64{}, 0
				s.Source = "mouse"
				if sawForce {
					s.Source = "trackpad"
				}
			})
			sample(ev, phase)
		case typePressure:
			sawForce = true
			p := float64(objc.Send[float32](ev, selPressure))
			update(func(s *State) { s.Pressure, s.Source = p, "trackpad" })
		case typeLeftMouseUp:
			buttonDown = false
			update(func(s *State) {
				s.Pressure, s.Tilt, s.Rotation = 0, [2]float64{}, 0
			})
			sample(ev, PhaseUp)
		}
		return ev
	})
	mask := uint64(1<<typeLeftMouseDown | 1<<typeLeftMouseUp | 1<<typeLeftMouseDragged |
		1<<typeTabletPoint | 1<<typeTabletProximity | 1<<typePressure)
	monitor = objc.ID(objc.GetClass("NSEvent")).Send(
		objc.RegisterName("addLocalMonitorForEventsMatchingMask:handler:"), mask, handler)
}

// readTablet stores pressure, tilt and rotation from a tablet event.
func readTablet(ev objc.ID, selPressure, selTilt, selRotation objc.SEL) {
	p := float64(objc.Send[float32](ev, selPressure))
	tl := objc.Send[nsPoint](ev, selTilt)
	rot := float64(objc.Send[float32](ev, selRotation))
	update(func(s *State) {
		s.Pressure, s.Tilt, s.Rotation, s.Source = p, [2]float64{tl.X, tl.Y}, rot, "tablet"
	})
}

// systemUptime is NSProcessInfo's systemUptime: the clock event timestamps use.
func systemUptime() float64 {
	info := objc.ID(objc.GetClass("NSProcessInfo")).Send(objc.RegisterName("processInfo"))
	return objc.Send[float64](info, objc.RegisterName("systemUptime"))
}
