package pen

// What the window procedure does with a Windows Ink pointer message, apart
// from the calls that read it. pen_windows.go gives the real calls; the tests
// give fake ones, so this runs on every OS.

// Windows pointer types (POINTER_INPUT_TYPE).
const (
	ptTouch    = 2 // PT_TOUCH, a finger on a touch screen
	ptPen      = 3 // PT_PEN
	ptMouse    = 4 // PT_MOUSE, only with EnableMouseInPointer
	ptTouchpad = 5 // PT_TOUCHPAD
)

// inkSystem is the part of Windows the pointer messages need.
type inkSystem interface {
	// pointerType is the type of a pointer id; false when it is unknown.
	pointerType(id uint32) (uint32, bool)
	// penHistory is every pen frame since the last message, newest first,
	// with X and Y in client pixels.
	penHistory(id uint32) []penRecord
	dpi() uint32  // the window's dpi
	tick() uint32 // GetTickCount, read now
}

// inkSource is the source a pointer type reports as. Only a pen has
// pressure: touch screens and touchpads send none to a normal window, so
// they count as a mouse, like the mouse messages Windows makes from them.
func inkSource(ptype uint32) string {
	if ptype == ptPen {
		return "tablet"
	}
	return "mouse"
}

// inkPointer handles WM_POINTERDOWN, WM_POINTERUPDATE and WM_POINTERUP: it
// records every pen frame the message stands for, oldest first. A finger
// or another pointer only stops the pen's reading from being used.
func inkPointer(sys inkSystem, id uint32, now float64) {
	ptype, ok := sys.pointerType(id)
	if !ok {
		return
	}
	if inkSource(ptype) != "tablet" {
		forgetPen()
		return
	}
	recs := sys.penHistory(id)
	if len(recs) == 0 {
		return
	}
	oldestFirst(recs)
	dpi, tick := sys.dpi(), sys.tick()
	for _, r := range recs {
		s := penSample(r, dpi, now, tick)
		eraser := penEraser(r.PenFlags)
		record(s)
		update(func(st *State) {
			st.Pressure, st.Tilt, st.Rotation = s.Pressure, s.Tilt, s.Rotation
			st.Source, st.Eraser = "tablet", eraser
		})
	}
}

// inkLeave handles WM_POINTERLEAVE: a pen that leaves the window or its
// range is not touching anything any more.
func inkLeave(sys inkSystem, id uint32) {
	if ptype, ok := sys.pointerType(id); ok && ptype == ptPen {
		update(func(s *State) { *s = State{OK: true} })
	}
}
