package pen

import (
	"math"
	"testing"
)

// fakeX is an xServer with made-up devices and a cursor the test moves.
type fakeX struct {
	devs []xDevice
	at   xPointer
	ok   bool
}

func (f *fakeX) devices() []xDevice             { return f.devs }
func (f *fakeX) pointer(int32) (xPointer, bool) { return f.at, f.ok }
func (f *fakeX) moveTo(x, y float64, inside bool) {
	f.at, f.ok = xPointer{X: x, Y: y, Inside: inside}, true
}

// A wacom-like pen: x, y, pressure 0..2047, tilt -64..63.
var (
	xPen = newXDevice(10, 2, "Wacom Intuos Pen stylus", map[string]xAxis{
		"Abs X": {0, 0, 15200}, "Abs Y": {1, 0, 9500},
		labelPressure: {2, 0, 2047}, labelTiltX: {3, -64, 63}, labelTiltY: {4, -64, 63},
	})
	xEraser = newXDevice(11, 2, "Wacom Intuos Pen eraser", map[string]xAxis{labelPressure: {2, 0, 2047}})
	xMouse  = newXDevice(12, 2, "Logitech USB Optical Mouse", map[string]xAxis{"Rel X": {0, -1, -1}, "Rel Y": {1, -1, -1}})
)

func TestDecodeValuators(t *testing.T) {
	for _, c := range []struct {
		mask   []byte
		values []float64
		want   map[int]float64
	}{
		{nil, nil, map[int]float64{}},
		{[]byte{0b00000101}, []float64{7, 9}, map[int]float64{0: 7, 2: 9}},
		{[]byte{0, 0b10}, []float64{3}, map[int]float64{9: 3}},
		{[]byte{0b111}, []float64{1}, map[int]float64{0: 1}}, // fewer values than bits
	} {
		got := decodeValuators(c.mask, c.values)
		if len(got) != len(c.want) {
			t.Errorf("%v %v: %v, want %v", c.mask, c.values, got, c.want)
			continue
		}
		for k, v := range c.want {
			if got[k] != v {
				t.Errorf("%v %v: %v, want %v", c.mask, c.values, got, c.want)
			}
		}
	}
}

func TestXDeviceKinds(t *testing.T) {
	for _, c := range []struct {
		d              xDevice
		tablet, eraser bool
	}{
		{xPen, true, false}, {xEraser, true, true}, {xMouse, false, false},
		{newXDevice(13, 2, "HUION Kamvas Pen (0)", map[string]xAxis{labelPressure: {2, 0, 65535}}), true, false},
		{newXDevice(14, 2, "HUION Kamvas Eraser (0)", map[string]xAxis{labelPressure: {2, 0, 65535}}), true, true},
		{newXDevice(15, 2, "xwayland-tablet eraser:22", map[string]xAxis{labelPressure: {2, 0, 65535}}), true, true},
		{newXDevice(16, 2, "xwayland-pointer:22", nil), false, false},
	} {
		if c.d.Tablet() != c.tablet || c.d.Eraser() != c.eraser {
			t.Errorf("%s: tablet %v eraser %v", c.d.Name, c.d.Tablet(), c.d.Eraser())
		}
	}
}

func TestXSampleRanges(t *testing.T) {
	last := map[int]float64{2: 2047, 3: -64, 4: 63}
	s := xSample(xPen, last, PhaseDrag)
	if !near(s.Pressure, 1) || !near(s.Tilt[0], -1) || !near(s.Tilt[1], 1) || s.Source != "tablet" {
		t.Errorf("full = %+v", s)
	}
	last = map[int]float64{2: 3000, 3: -0.5, 4: -0.5}
	if s := xSample(xPen, last, PhaseDrag); !near(s.Pressure, 1) || !near(s.Tilt[0], 0) {
		t.Errorf("clamped and middle = %+v", s)
	}
	if s := xSample(xPen, map[int]float64{}, PhaseDown); s.Pressure != 1 || s.Tilt != [2]float64{} {
		t.Errorf("no axes yet = %+v", s)
	}
	for _, phase := range []string{PhaseUp, PhasePen} {
		if s := xSample(xPen, map[int]float64{2: 1000}, phase); s.Pressure != 0 {
			t.Errorf("%s pressure = %v", phase, s.Pressure)
		}
	}
	rot := newXDevice(1, 2, "art pen", map[string]xAxis{labelPressure: {2, 0, 1}, labelRotation: {5, 0, 100}})
	if s := xSample(rot, map[int]float64{5: 25}, PhaseDrag); !near(s.Rotation, 90) {
		t.Errorf("rotation = %v", s.Rotation)
	}
}

func TestXReaderStroke(t *testing.T) {
	resetPen()
	x := &fakeX{devs: []xDevice{xPen, xMouse}}
	r := newXReader(x, 2) // Xft.dpi 192
	x.moveTo(40, 60, true)
	r.handle(xRaw{Type: xiRawMotion, Device: 10, Time: 1000, Values: map[int]float64{2: 0, 3: 0}}, 5)
	r.handle(xRaw{Type: xiRawButtonPress, Device: 10, Detail: 1, Time: 1010, Values: map[int]float64{2: 511.75}}, 5.01)
	x.moveTo(50, 70, true)
	// Only the pressure changes: the tilt keeps its last value.
	r.handle(xRaw{Type: xiRawMotion, Device: 10, Time: 1020, Values: map[int]float64{2: 1535.25}}, 5.03)
	r.handle(xRaw{Type: xiRawButtonPress, Device: 10, Detail: 2, Time: 1025}, 5.03) // a barrel button
	r.handle(xRaw{Type: xiRawButtonRelease, Device: 10, Detail: 1, Time: 1030}, 5.04)
	got := Drain()
	phases := []string{PhasePen, PhaseDown, PhaseDrag, PhaseUp}
	if len(got) != len(phases) {
		t.Fatalf("samples = %+v", got)
	}
	for i, s := range got {
		if s.Phase != phases[i] || s.Source != "tablet" {
			t.Errorf("sample %d = %+v, want %s", i, s, phases[i])
		}
	}
	if d := got[2]; !near(d.Pressure, 0.75) || d.X != 25 || d.Y != 35 || !near(d.Tilt[0], 64.0/127*2-1) || !near(d.T, 5.02) {
		t.Errorf("drag = %+v", d)
	}
	if !near(got[1].Pressure, 0.25) || got[3].Pressure != 0 {
		t.Errorf("down %v up %v", got[1].Pressure, got[3].Pressure)
	}
	if s := Latest(); s.Source != "tablet" || s.Eraser {
		t.Errorf("state = %+v", s)
	}
}

func TestXReaderOutsideAndOthers(t *testing.T) {
	resetPen()
	x := &fakeX{devs: []xDevice{xPen, xEraser, xMouse}}
	r := newXReader(x, 1)
	// A stroke that begins outside the window is not the window's.
	x.moveTo(-5, 10, false)
	r.handle(xRaw{Type: xiRawMotion, Device: 10}, 0)
	r.handle(xRaw{Type: xiRawButtonPress, Device: 10, Detail: 1}, 0)
	x.moveTo(5, 10, true)
	r.handle(xRaw{Type: xiRawMotion, Device: 10}, 0)
	r.handle(xRaw{Type: xiRawButtonRelease, Device: 10, Detail: 1}, 0)
	if got := Drain(); len(got) != 1 || got[0].Phase != PhasePen {
		t.Errorf("outside stroke gave %+v", got)
	}
	// The eraser, named by the event's source rather than its device.
	r.handle(xRaw{Type: xiRawButtonPress, Device: 2, Source: 11, Detail: 1, Values: map[int]float64{2: 2047}}, 0)
	if s := Latest(); !s.Eraser || !near(s.Pressure, 1) {
		t.Errorf("eraser state = %+v", s)
	}
	// A mouse after the pen takes the reading back to a mouse, with no sample.
	Drain()
	r.handle(xRaw{Type: xiRawMotion, Device: 12}, 0)
	if s := Latest(); s.Source != "mouse" || s.Eraser {
		t.Errorf("after the mouse, state = %+v", s)
	}
	// Unknown devices and an unknown cursor place give nothing.
	r.handle(xRaw{Type: xiRawMotion, Device: 99}, 0)
	x.ok = false
	r.handle(xRaw{Type: xiRawMotion, Device: 10}, 0)
	if got := Drain(); len(got) != 0 {
		t.Errorf("samples = %+v", got)
	}
}

func TestXReaderRefresh(t *testing.T) {
	resetPen()
	x := &fakeX{devs: []xDevice{xMouse}}
	r := newXReader(x, 1)
	x.moveTo(1, 1, true)
	r.handle(xRaw{Type: xiRawMotion, Device: 10}, 0)
	if len(Drain()) != 0 {
		t.Fatal("a pen not yet plugged in made a sample")
	}
	x.devs = append(x.devs, xPen) // plugged in: XI_HierarchyChanged
	r.refresh()
	r.handle(xRaw{Type: xiRawMotion, Device: 10}, 0)
	if len(Drain()) != 1 {
		t.Error("the new pen is not read")
	}
}

func TestXClock(t *testing.T) {
	var c xClock
	steps := []struct {
		now  float64
		ms   uint32
		want float64
	}{
		{10, 5000, 10},         // the first sets the offset
		{10.05, 5020, 10.02},   // read late: still on the server's clock
		{10.025, 5030, 10.025}, // read sooner than any before: a smaller offset
		{10.04, 5040, 10.035},
	}
	for i, s := range steps {
		if got := c.at(s.now, s.ms); !near(got, s.want) {
			t.Errorf("step %d = %v, want %v", i, got, s.want)
		}
	}
	// The 32-bit millisecond clock wraps.
	var w xClock
	w.at(1, 0xFFFFFFF6)
	if got := w.at(1.02, 10); math.Abs(got-1.02) > 1e-6 {
		t.Errorf("wrapped = %v", got)
	}
}

func TestXftScale(t *testing.T) {
	for _, c := range []struct {
		res  string
		want float64
	}{
		{"", 1},
		{"Xft.antialias:\t1\nXft.dpi:\t144\nXft.hinting:\t1\n", 1.5},
		{"Xft.dpi: 192", 2},
		{"Xft.dpi:\tlots", 1},
		{"Xcursor.size:\t24\n", 1},
	} {
		if got := xftScale(c.res); !near(got, c.want) {
			t.Errorf("xftScale(%q) = %v, want %v", c.res, got, c.want)
		}
	}
}
