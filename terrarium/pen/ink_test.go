package pen

import "testing"

// fakeInk is an inkSystem with made-up pointers.
type fakeInk struct {
	types   map[uint32]uint32
	history []penRecord // newest first, as Windows gives it
}

func (f *fakeInk) pointerType(id uint32) (uint32, bool) {
	t, ok := f.types[id]
	return t, ok
}

func (f *fakeInk) penHistory(uint32) []penRecord {
	return append([]penRecord(nil), f.history...)
}

func (f *fakeInk) dpi() uint32  { return 144 }
func (f *fakeInk) tick() uint32 { return 2000 }

// resetPen empties the reading and the queue between tests.
func resetPen() {
	mu.Lock()
	state, queue = State{}, nil
	mu.Unlock()
}

func TestInkSource(t *testing.T) {
	for _, c := range []struct {
		ptype uint32
		want  string
	}{{ptPen, "tablet"}, {ptTouch, "mouse"}, {ptMouse, "mouse"}, {ptTouchpad, "mouse"}, {1, "mouse"}} {
		if got := inkSource(c.ptype); got != c.want {
			t.Errorf("inkSource(%d) = %s, want %s", c.ptype, got, c.want)
		}
	}
}

func TestInkPointerRecordsHistoryOldestFirst(t *testing.T) {
	resetPen()
	sys := &fakeInk{types: map[uint32]uint32{1: ptPen}, history: []penRecord{
		{PointerFlags: flagInContact, PenMask: penMaskPressure, Pressure: 768, X: 30, Y: 30, Time: 2000},
		{PointerFlags: flagInContact | flagDown, PenMask: penMaskPressure, Pressure: 256, X: 15, Y: 15, Time: 1990},
	}}
	inkPointer(sys, 1, 10)
	got := Drain()
	if len(got) != 2 {
		t.Fatalf("samples = %+v", got)
	}
	if got[0].Phase != PhaseDown || !near(got[0].Pressure, 0.25) || got[0].X != 10 || !near(got[0].T, 9.99) {
		t.Errorf("first = %+v", got[0])
	}
	if got[1].Phase != PhaseDrag || !near(got[1].Pressure, 0.75) || got[1].X != 20 || !near(got[1].T, 10) {
		t.Errorf("second = %+v", got[1])
	}
	if s := Latest(); !s.OK || s.Source != "tablet" || !near(s.Pressure, 0.75) || s.Eraser {
		t.Errorf("state = %+v", s)
	}
}

func TestInkPointerEraser(t *testing.T) {
	resetPen()
	sys := &fakeInk{types: map[uint32]uint32{1: ptPen}, history: []penRecord{
		{PointerFlags: flagInContact, PenFlags: penFlagEraser, PenMask: penMaskPressure, Pressure: 512},
	}}
	inkPointer(sys, 1, 0)
	if s := Latest(); !s.Eraser {
		t.Errorf("state = %+v, want the eraser", s)
	}
}

func TestInkTouchForgetsThePen(t *testing.T) {
	resetPen()
	sys := &fakeInk{types: map[uint32]uint32{1: ptPen, 2: ptTouch}, history: []penRecord{
		{PenMask: penMaskPressure, Pressure: 900}, // hovering
	}}
	inkPointer(sys, 1, 0)
	if Latest().Source != "tablet" {
		t.Fatal("the pen should be read")
	}
	Drain()
	inkPointer(sys, 2, 0)
	if s := Latest(); s.Source != "mouse" {
		t.Errorf("after a finger, state = %+v", s)
	}
	if got := Drain(); len(got) != 0 {
		t.Errorf("a finger made samples: %+v", got)
	}
}

func TestInkUnknownPointerAndLeave(t *testing.T) {
	resetPen()
	sys := &fakeInk{types: map[uint32]uint32{1: ptPen, 2: ptTouch}}
	inkPointer(sys, 9, 0) // unknown id
	inkPointer(sys, 1, 0) // a pen with no history
	if got := Drain(); len(got) != 0 || Latest().OK {
		t.Errorf("samples %+v, state %+v", got, Latest())
	}
	update(func(s *State) { s.Pressure, s.Source = 0.5, "tablet" })
	inkLeave(sys, 2)
	if Latest().Source != "tablet" {
		t.Error("a finger leaving should not reset the pen")
	}
	inkLeave(sys, 1)
	if s := Latest(); s.Source != "" || s.Pressure != 0 || !s.OK {
		t.Errorf("after the pen left, state = %+v", s)
	}
}
