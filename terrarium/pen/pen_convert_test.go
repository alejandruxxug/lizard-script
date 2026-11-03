package pen

import (
	"math"
	"testing"
)

func near(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestPenPressure(t *testing.T) {
	for _, c := range []struct {
		r    penRecord
		want float64
	}{
		{penRecord{PenMask: penMaskPressure, Pressure: 0}, 0},
		{penRecord{PenMask: penMaskPressure, Pressure: 512}, 0.5},
		{penRecord{PenMask: penMaskPressure, Pressure: 1024}, 1},
		{penRecord{PenMask: penMaskPressure, Pressure: 5000}, 1},
		{penRecord{PointerFlags: flagInContact}, 1},
		{penRecord{}, 0},
	} {
		if got := penPressure(c.r); !near(got, c.want) {
			t.Errorf("%+v: %v, want %v", c.r, got, c.want)
		}
	}
}

func TestPenTilt(t *testing.T) {
	for _, c := range []struct {
		deg  int32
		want float64
	}{{-90, -1}, {0, 0}, {45, 0.5}, {90, 1}, {120, 1}, {-200, -1}} {
		if got := penTilt(c.deg); !near(got, c.want) {
			t.Errorf("penTilt(%d) = %v, want %v", c.deg, got, c.want)
		}
	}
}

func TestPenEraserAndPhase(t *testing.T) {
	if penEraser(0) || !penEraser(penFlagEraser) || !penEraser(penFlagInverted) {
		t.Error("eraser flags")
	}
	for _, c := range []struct {
		f    uint32
		want string
	}{
		{flagDown | flagInContact, PhaseDown}, {flagInContact, PhaseDrag},
		{flagUp, PhaseUp}, {0, PhasePen},
	} {
		if got := penPhase(c.f); got != c.want {
			t.Errorf("penPhase(%#x) = %s, want %s", c.f, got, c.want)
		}
	}
}

func TestLogicalPoint(t *testing.T) {
	for _, c := range []struct {
		px, py int32
		dpi    uint32
		x, y   float64
	}{{200, 100, 96, 200, 100}, {300, 150, 144, 200, 100}, {192, 96, 192, 96, 48}, {10, 10, 0, 10, 10}} {
		if x, y := logicalPoint(c.px, c.py, c.dpi); !near(x, c.x) || !near(y, c.y) {
			t.Errorf("logicalPoint(%d,%d,%d) = %v,%v", c.px, c.py, c.dpi, x, y)
		}
	}
}

func TestSampleTimeWraps(t *testing.T) {
	if got := sampleTime(10, 1000, 990); !near(got, 9.99) {
		t.Errorf("earlier = %v", got)
	}
	// The tick counter wrapped between the event and now.
	if got := sampleTime(10, 5, 0xFFFFFFFB); !near(got, 9.99) {
		t.Errorf("wrapped = %v", got)
	}
}

func TestPenSample(t *testing.T) {
	r := penRecord{PointerFlags: flagInContact, PenMask: penMaskPressure | penMaskTiltX | penMaskTiltY | penMaskRotation,
		Pressure: 256, TiltX: 45, TiltY: -90, Rotation: 90, X: 288, Y: 144, Time: 1000}
	s := penSample(r, 144, 5, 1000)
	if s.X != 192 || s.Y != 96 || !near(s.Pressure, 0.25) || !near(s.Tilt[0], 0.5) || !near(s.Tilt[1], -1) ||
		s.Rotation != 90 || s.Source != "tablet" || s.Phase != PhaseDrag || !near(s.T, 5) {
		t.Errorf("sample = %+v", s)
	}
	if up := penSample(penRecord{PointerFlags: flagUp, PenMask: penMaskPressure, Pressure: 900}, 96, 0, 0); up.Pressure != 0 {
		t.Errorf("up pressure = %v", up.Pressure)
	}
}

func TestOldestFirst(t *testing.T) {
	rs := []penRecord{{Time: 3}, {Time: 2}, {Time: 1}, {Time: 0}}
	oldestFirst(rs)
	for i, r := range rs {
		if r.Time != uint32(i) {
			t.Fatalf("order %v", rs)
		}
	}
	oldestFirst(nil)
}
