package pen

import (
	"strconv"
	"strings"
)

// What the X11 reader does with an XInput2 raw event, apart from the calls
// that read it. pen_linux.go gives the real calls; the tests give fake ones,
// so this runs on every OS.
//
// Raw events come from every device to every client that asks for them on
// the root window, next to the normal events the window gets, so reading
// them changes nothing for Ebitengine. They carry the device's axes but not
// the pointer's place in the window, which the reader asks the server for.

// XInput2 raw event types.
const (
	xiRawButtonPress   = 15
	xiRawButtonRelease = 16
	xiRawMotion        = 17
)

// The tip of a pen is button 1 under every X driver (wacom, libinput,
// XWayland); the barrel buttons are 2 and up.
const xTipButton = 1

// xAxis is one valuator of a device: its number and range. Number is -1
// when the device doesn't have the axis.
type xAxis struct {
	Number   int
	Min, Max float64
}

var noAxis = xAxis{Number: -1}

// unit maps a value of the axis to 0..1.
func (a xAxis) unit(v float64) float64 {
	if a.Max <= a.Min {
		return clamp01(v)
	}
	return clamp01((v - a.Min) / (a.Max - a.Min))
}

// xDevice is a slave pointer of the X server, as XIQueryDevice tells it.
type xDevice struct {
	ID, Master int32 // Master is the cursor it moves, 0 when floating
	Name       string
	Pressure   xAxis
	TiltX      xAxis
	TiltY      xAxis
	Rotation   xAxis
}

// The valuator labels of the axes, as every X input driver names them.
const (
	labelPressure = "Abs Pressure"
	labelTiltX    = "Abs Tilt X"
	labelTiltY    = "Abs Tilt Y"
	labelRotation = "Abs Rz"
)

// newXDevice builds a device from its valuators, given by label.
func newXDevice(id, master int32, name string, axes map[string]xAxis) xDevice {
	axis := func(label string) xAxis {
		if a, ok := axes[label]; ok {
			return a
		}
		return noAxis
	}
	return xDevice{ID: id, Master: master, Name: name, Pressure: axis(labelPressure),
		TiltX: axis(labelTiltX), TiltY: axis(labelTiltY), Rotation: axis(labelRotation)}
}

// Tablet tells a pen from a mouse: only a pen has a pressure axis.
func (d xDevice) Tablet() bool { return d.Pressure.Number >= 0 }

// Eraser tells the eraser end of a pen: the drivers make it a device of its
// own, named "... eraser" (wacom, XWayland) or "... Eraser (0)" (libinput).
func (d xDevice) Eraser() bool { return strings.Contains(strings.ToLower(d.Name), "eraser") }

// xRaw is one raw event reduced to what the reader uses.
type xRaw struct {
	Type           int
	Device, Source int32
	Detail         int32 // the button of a press or release
	Time           uint32
	Values         map[int]float64 // the axes the event carries, by number
}

// decodeValuators pairs the set bits of an XIValuatorState mask with its
// values, which only list the axes that changed, lowest number first.
func decodeValuators(mask []byte, values []float64) map[int]float64 {
	out := map[int]float64{}
	k := 0
	for i := 0; i < len(mask)*8 && k < len(values); i++ {
		if mask[i/8]&(1<<(i%8)) != 0 {
			out[i] = values[k]
			k++
		}
	}
	return out
}

// xPointer is where a device's cursor is, in window pixels from the top
// left, and whether that is inside the window.
type xPointer struct {
	X, Y   float64
	Inside bool
}

// xServer is the part of the X server the reader needs.
type xServer interface {
	// devices lists the slave pointers.
	devices() []xDevice
	// pointer is where the cursor master is; false when it can't be told
	// (another screen, the window is gone).
	pointer(master int32) (xPointer, bool)
}

// xReader turns raw events into samples and readings.
type xReader struct {
	server xServer
	scale  float64 // window pixels per logical point
	devs   map[int32]xDevice
	values map[int32]map[int]float64 // the last value of each axis, per device
	down   map[int32]bool            // the tip touches, in a stroke that began inside the window
	clock  xClock
}

func newXReader(server xServer, scale float64) *xReader {
	r := &xReader{server: server, scale: scale, values: map[int32]map[int]float64{}, down: map[int32]bool{}}
	r.refresh()
	return r
}

// refresh reads the devices again, after one was plugged in or out.
func (r *xReader) refresh() {
	r.devs = map[int32]xDevice{}
	for _, d := range r.server.devices() {
		r.devs[d.ID] = d
	}
}

// device is the slave an event came from. Depending on the server, a raw
// event names it as the device or as the source.
func (r *xReader) device(ev xRaw) (xDevice, bool) {
	if d, ok := r.devs[ev.Device]; ok {
		return d, true
	}
	d, ok := r.devs[ev.Source]
	return d, ok
}

// handle takes one raw event; now is Now() as it is read.
func (r *xReader) handle(ev xRaw, now float64) {
	d, ok := r.device(ev)
	if !ok {
		return
	}
	if !d.Tablet() {
		forgetPen()
		return
	}
	last := r.values[d.ID]
	if last == nil {
		last = map[int]float64{}
		r.values[d.ID] = last
	}
	for k, v := range ev.Values {
		last[k] = v
	}
	if (ev.Type == xiRawButtonPress || ev.Type == xiRawButtonRelease) && ev.Detail != xTipButton {
		return
	}
	at, ok := r.server.pointer(d.Master)
	if !ok {
		return
	}
	phase := r.phase(d.ID, ev.Type, at.Inside)
	if phase == "" {
		return
	}
	s := xSample(d, last, phase)
	s.T = r.clock.at(now, ev.Time)
	s.X, s.Y = at.X/r.scale, at.Y/r.scale
	record(s)
	eraser := d.Eraser()
	update(func(st *State) {
		st.Pressure, st.Tilt, st.Rotation = s.Pressure, s.Tilt, s.Rotation
		st.Source, st.Eraser = "tablet", eraser
	})
}

// phase names the gesture part of an event, or "" when it is none of the
// window's business: a stroke or a hover outside the window.
func (r *xReader) phase(id int32, kind int, inside bool) string {
	switch kind {
	case xiRawButtonPress:
		if !inside {
			return ""
		}
		r.down[id] = true
		return PhaseDown
	case xiRawButtonRelease:
		if !r.down[id] {
			return ""
		}
		r.down[id] = false
		return PhaseUp
	}
	if r.down[id] {
		return PhaseDrag
	}
	if inside {
		return PhasePen
	}
	return ""
}

// xSample is the pressure, tilt and rotation of a device, from the last
// value of each axis. A pen that never sent its pressure presses fully.
func xSample(d xDevice, last map[int]float64, phase string) Sample {
	get := func(a xAxis, otherwise float64) float64 {
		if v, ok := last[a.Number]; ok && a.Number >= 0 {
			return a.unit(v)
		}
		return otherwise
	}
	p := get(d.Pressure, 1)
	if phase == PhaseUp || phase == PhasePen {
		p = 0
	}
	return Sample{
		Pressure: p,
		Tilt:     [2]float64{get(d.TiltX, 0.5)*2 - 1, get(d.TiltY, 0.5)*2 - 1},
		Rotation: get(d.Rotation, 0) * 360,
		Source:   "tablet", Phase: phase,
	}
}

// xClock places the X server's millisecond timestamps on the app clock.
// The server's clock starts at its own time, so the offset is learned: it
// is the smallest delay seen, which keeps every sample at or before the
// moment it was read. The 32-bit timestamp may wrap.
type xClock struct {
	started bool
	last    uint32
	ms      int64 // last, with the wraps added
	offset  float64
}

func (c *xClock) at(now float64, t uint32) float64 {
	if !c.started {
		c.started, c.ms, c.offset = true, int64(t), now-float64(t)/1000
	} else {
		c.ms += int64(int32(t - c.last))
	}
	c.last = t
	if o := now - float64(c.ms)/1000; o < c.offset {
		c.offset = o
	}
	return float64(c.ms)/1000 + c.offset
}

// xftScale is the window pixels per logical point Ebitengine uses on X11:
// the Xft.dpi of the server's resources over 96, or 1 when it is not set.
func xftScale(resources string) float64 {
	for _, line := range strings.Split(resources, "\n") {
		key, value, ok := strings.Cut(line, ":")
		if !ok || strings.TrimSpace(key) != "Xft.dpi" {
			continue
		}
		if dpi, err := strconv.ParseFloat(strings.TrimSpace(value), 64); err == nil && dpi > 0 {
			return dpi / 96
		}
	}
	return 1
}
