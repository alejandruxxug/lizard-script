//go:build linux

package pen

import (
	"os"
	"runtime"
	"structs"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

// XInput2 on X11, and so on XWayland too: Ebitengine's window is an X11
// window everywhere on Linux. No cgo: libX11 and libXi are loaded with
// purego, like Ebitengine loads them. The reader keeps a connection of its
// own and asks for raw events on the root window, which the server sends to
// every client that wants them, beside the window's normal events; nothing
// Ebitengine receives changes. What an event means is decided in xinput.go;
// this file only makes the calls.

// X11 and XInput2 values.
const (
	genericEvent    = 35
	destroyNotify   = 17
	configureNotify = 22

	structureNotifyMask = 1 << 17

	xiAllDevices       = 0
	xiDeviceChanged    = 1
	xiHierarchyChanged = 11
	xiSlavePointer     = 3
	xiValuatorType     = 2

	xaCardinal = 6 // XA_CARDINAL
)

// xEvent is an XEvent: a union of 24 longs (Go's int is a C long on Linux).
type xEvent struct {
	_    structs.HostLayout
	data [24]int
}

// xCookie is an XGenericEventCookie.
type xCookie struct {
	_         structs.HostLayout
	Type      int32
	Serial    uint
	SendEvent int32
	Display   uintptr
	Extension int32
	Evtype    int32
	Cookie    uint32
	Data      unsafe.Pointer
}

// xWindowEvent is the start of XDestroyWindowEvent and XConfigureEvent.
type xWindowEvent struct {
	_             structs.HostLayout
	Type          int32
	Serial        uint
	SendEvent     int32
	Display       uintptr
	Event, Window uint
	X, Y, W, H    int32 // only in a ConfigureNotify
}

// xiRawEvent is an XIRawEvent.
type xiRawEvent struct {
	_         structs.HostLayout
	Type      int32
	Serial    uint
	SendEvent int32
	Display   uintptr
	Extension int32
	Evtype    int32
	Time      uint
	Deviceid  int32
	Sourceid  int32
	Detail    int32
	Flags     int32
	MaskLen   int32
	Mask      *byte
	Values    *float64
	RawValues *float64
}

// xiEventMask is an XIEventMask.
type xiEventMask struct {
	_        structs.HostLayout
	Deviceid int32
	MaskLen  int32
	Mask     *byte
}

// xiDeviceInfo is an XIDeviceInfo.
type xiDeviceInfo struct {
	_          structs.HostLayout
	Deviceid   int32
	Name       *byte
	Use        int32
	Attachment int32
	Enabled    int32
	NumClasses int32
	Classes    **xiValuatorClass
}

// xiValuatorClass is an XIValuatorClassInfo; its first two fields are the
// XIAnyClassInfo every class starts with.
type xiValuatorClass struct {
	_          structs.HostLayout
	Type       int32
	Sourceid   int32
	Number     int32
	Label      uint
	Min, Max   float64
	Value      float64
	Resolution int32
	Mode       int32
}

// xiButtonState and xiModState are XIButtonState and XIModifierState.
type xiButtonState struct {
	_       structs.HostLayout
	MaskLen int32
	Mask    unsafe.Pointer
}

type xiModState struct {
	_                                structs.HostLayout
	Base, Latched, Locked, Effective int32
}

var (
	xInitThreads           func() int32
	xOpenDisplay           func(name *byte) uintptr
	xDefaultRootWindow     func(d uintptr) uint
	xInternAtom            func(d uintptr, name string, onlyIfExists int32) uint
	xGetAtomName           func(d uintptr, atom uint) *byte
	xFree                  func(p unsafe.Pointer) int32
	xQueryExtension        func(d uintptr, name string, opcode, event, errorBase *int32) int32
	xQueryTree             func(d uintptr, w uint, root, parent *uint, children **uint, n *uint32) int32
	xGetWindowProperty     func(d uintptr, w, prop uint, offset, length int, del int32, reqType uint, actualType *uint, actualFormat *int32, nitems, bytesAfter *uint, data **byte) int32
	xGetWindowAttributes   func(d uintptr, w uint, attrs *xEvent) int32
	xSelectInput           func(d uintptr, w uint, mask int) int32
	xNextEvent             func(d uintptr, ev *xEvent) int32
	xGetEventData          func(d uintptr, c *xCookie) int32
	xFreeEventData         func(d uintptr, c *xCookie)
	xResourceManagerString func(d uintptr) *byte
	xSetErrorHandler       func(handler uintptr) uintptr

	xiQueryVersion   func(d uintptr, major, minor *int32) int32
	xiSelectEvents   func(d uintptr, w uint, masks *xiEventMask, n int32) int32
	xiQueryDevice    func(d uintptr, id int32, n *int32) *xiDeviceInfo
	xiFreeDeviceInfo func(info *xiDeviceInfo)
	xiQueryPointer   func(d uintptr, id int32, w uint, root, child *uint, rootX, rootY, winX, winY *float64, buttons *xiButtonState, mods, group *xiModState) int32
)

// x11 is the reader's connection, and the real xServer.
type x11 struct {
	d      uintptr
	root   uint
	opcode int32 // XInput's major opcode: its events carry it
	win    uint
	w, h   int32 // the window's size, kept up by ConfigureNotify
}

var (
	probeOnce sync.Once
	conn      *x11 // nil when the system has no XInput2
	startOnce sync.Once
	installed bool

	// errorCallback and errorNext chain the X error handler: an error on
	// the reader's connection (the window closed while asking where the
	// pointer is) is dropped, and any other goes to the handler before.
	errorCallback uintptr
	errorNext     uintptr
)

// Supported reports whether the X server has XInput 2.1 or later: from 2.1
// on, raw events keep coming while a button is held. It connects once.
func Supported() bool {
	probeOnce.Do(probe)
	return conn != nil
}

// Start finds the app's window and starts reading raw events. Call it once
// after the window exists (Ebitengine's first Update is fine). It reports
// whether the reader runs.
func Start() bool {
	startOnce.Do(install)
	return installed
}

// probe loads the libraries and opens the reader's connection.
func probe() {
	x11lib, err := purego.Dlopen("libX11.so.6", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
	if err != nil {
		return
	}
	xilib, err := purego.Dlopen("libXi.so.6", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
	if err != nil {
		return
	}
	x11funcs := map[string]any{
		"XInitThreads": &xInitThreads, "XOpenDisplay": &xOpenDisplay, "XDefaultRootWindow": &xDefaultRootWindow,
		"XInternAtom": &xInternAtom, "XGetAtomName": &xGetAtomName, "XFree": &xFree,
		"XQueryExtension": &xQueryExtension, "XQueryTree": &xQueryTree, "XGetWindowProperty": &xGetWindowProperty,
		"XGetWindowAttributes": &xGetWindowAttributes, "XSelectInput": &xSelectInput, "XNextEvent": &xNextEvent,
		"XGetEventData": &xGetEventData, "XFreeEventData": &xFreeEventData,
		"XResourceManagerString": &xResourceManagerString, "XSetErrorHandler": &xSetErrorHandler,
	}
	xifuncs := map[string]any{
		"XIQueryVersion": &xiQueryVersion, "XISelectEvents": &xiSelectEvents, "XIQueryDevice": &xiQueryDevice,
		"XIFreeDeviceInfo": &xiFreeDeviceInfo, "XIQueryPointer": &xiQueryPointer,
	}
	if !register(x11lib, x11funcs) || !register(xilib, xifuncs) {
		return
	}
	xInitThreads() // Ebitengine does the same; a second call does nothing
	d := xOpenDisplay(nil)
	if d == 0 {
		return
	}
	var opcode, event, errorBase int32
	if xQueryExtension(d, "XInputExtension", &opcode, &event, &errorBase) == 0 {
		return
	}
	major, minor := int32(2), int32(2)
	if xiQueryVersion(d, &major, &minor) != 0 || major < 2 || (major == 2 && minor < 1) {
		return
	}
	conn = &x11{d: d, root: xDefaultRootWindow(d), opcode: opcode}
}

// register binds the functions of a library, or none when one is missing
// (an old libXi): RegisterLibFunc would panic.
func register(lib uintptr, funcs map[string]any) bool {
	for name := range funcs {
		if _, err := purego.Dlsym(lib, name); err != nil {
			return false
		}
	}
	for name, fn := range funcs {
		purego.RegisterLibFunc(fn, lib, name)
	}
	return true
}

func install() {
	if !Supported() {
		return
	}
	c := conn
	c.win = c.findWindow(c.root, 0)
	if c.win == 0 {
		return
	}
	var attrs xEvent
	if xGetWindowAttributes(c.d, c.win, &attrs) != 0 {
		ints := (*[4]int32)(unsafe.Pointer(&attrs))
		c.w, c.h = ints[2], ints[3]
	}
	errorCallback = purego.NewCallback(func(d, ev uintptr) uintptr {
		if d == c.d || errorNext == 0 {
			return 0
		}
		r, _, _ := purego.SyscallN(errorNext, d, ev)
		return r
	})
	errorNext = xSetErrorHandler(errorCallback)
	xSelectInput(c.d, c.win, structureNotifyMask)
	mask := make([]byte, 4)
	for _, e := range []int{xiRawMotion, xiRawButtonPress, xiRawButtonRelease, xiHierarchyChanged, xiDeviceChanged} {
		mask[e/8] |= 1 << (e % 8)
	}
	xiSelectEvents(c.d, c.root, &xiEventMask{Deviceid: xiAllDevices, MaskLen: int32(len(mask)), Mask: &mask[0]}, 1)
	runtime.KeepAlive(mask)
	r := newXReader(c, xftScale(goString(xResourceManagerString(c.d))))
	installed = true
	go c.loop(r)
}

// findWindow looks for the window whose _NET_WM_PID is this process, under
// w and up to three levels down (a window manager puts it in a frame).
// Among several, the largest wins.
func (c *x11) findWindow(w uint, depth int) uint {
	pidAtom := xInternAtom(c.d, "_NET_WM_PID", 1)
	if pidAtom == 0 {
		return 0
	}
	var best uint
	var bestArea int32
	var walk func(w uint, depth int)
	walk = func(w uint, depth int) {
		var root, parent uint
		var children *uint
		var n uint32
		if xQueryTree(c.d, w, &root, &parent, &children, &n) == 0 || children == nil {
			return
		}
		list := append([]uint(nil), unsafe.Slice(children, n)...)
		xFree(unsafe.Pointer(children))
		for _, child := range list {
			if c.pidOf(child, pidAtom) == os.Getpid() {
				var attrs xEvent
				if xGetWindowAttributes(c.d, child, &attrs) != 0 {
					ints := (*[4]int32)(unsafe.Pointer(&attrs))
					if area := ints[2] * ints[3]; area > bestArea {
						best, bestArea = child, area
					}
				}
				continue
			}
			if depth < 3 {
				walk(child, depth+1)
			}
		}
	}
	walk(w, depth)
	return best
}

// pidOf reads a window's _NET_WM_PID, or -1.
func (c *x11) pidOf(w, atom uint) int {
	var actual, nitems, after uint
	var format int32
	var data *byte
	if xGetWindowProperty(c.d, w, atom, 0, 1, 0, xaCardinal, &actual, &format, &nitems, &after, &data) != 0 || data == nil {
		return -1
	}
	defer xFree(unsafe.Pointer(data))
	if actual != xaCardinal || format != 32 || nitems < 1 {
		return -1
	}
	// A format-32 property is a list of C longs.
	return int(uint32(*(*uint)(unsafe.Pointer(data))))
}

// loop reads the connection's events until the window is destroyed.
func (c *x11) loop(r *xReader) {
	runtime.LockOSThread()
	var ev xEvent
	for {
		xNextEvent(c.d, &ev)
		switch kind := *(*int32)(unsafe.Pointer(&ev)); kind {
		case genericEvent:
			c.generic(r, (*xCookie)(unsafe.Pointer(&ev)))
		case configureNotify:
			if we := (*xWindowEvent)(unsafe.Pointer(&ev)); we.Window == c.win {
				c.w, c.h = we.W, we.H
			}
		case destroyNotify:
			if we := (*xWindowEvent)(unsafe.Pointer(&ev)); we.Window == c.win {
				c.win = 0
				forgetPen()
				return
			}
		}
	}
}

// generic hands an XInput event to the reader.
func (c *x11) generic(r *xReader, cookie *xCookie) {
	if cookie.Extension != c.opcode || xGetEventData(c.d, cookie) == 0 {
		return
	}
	defer xFreeEventData(c.d, cookie)
	switch cookie.Evtype {
	case xiHierarchyChanged, xiDeviceChanged:
		r.refresh()
	case xiRawMotion, xiRawButtonPress, xiRawButtonRelease:
		raw := (*xiRawEvent)(cookie.Data)
		var mask []byte
		var values []float64
		if raw.MaskLen > 0 && raw.Mask != nil {
			mask = unsafe.Slice(raw.Mask, raw.MaskLen)
			n := 0
			for _, b := range mask {
				for ; b != 0; b &= b - 1 {
					n++
				}
			}
			if n > 0 && raw.Values != nil {
				values = unsafe.Slice(raw.Values, n)
			}
		}
		r.handle(xRaw{Type: int(cookie.Evtype), Device: raw.Deviceid, Source: raw.Sourceid,
			Detail: raw.Detail, Time: uint32(raw.Time), Values: decodeValuators(mask, values)}, Now())
	}
}

// devices lists the slave pointers and their axes.
func (c *x11) devices() []xDevice {
	var n int32
	info := xiQueryDevice(c.d, xiAllDevices, &n)
	if info == nil {
		return nil
	}
	defer xiFreeDeviceInfo(info)
	var out []xDevice
	for _, d := range unsafe.Slice(info, n) {
		if d.Use != xiSlavePointer || d.Classes == nil {
			continue
		}
		axes := map[string]xAxis{}
		for _, class := range unsafe.Slice(d.Classes, d.NumClasses) {
			if class == nil || class.Type != xiValuatorType || class.Label == 0 {
				continue
			}
			name := xGetAtomName(c.d, class.Label)
			if name == nil {
				continue
			}
			axes[goString(name)] = xAxis{Number: int(class.Number), Min: class.Min, Max: class.Max}
			xFree(unsafe.Pointer(name))
		}
		out = append(out, newXDevice(d.Deviceid, d.Attachment, goString(d.Name), axes))
	}
	return out
}

// pointer asks where the cursor of master is, in the window's pixels.
func (c *x11) pointer(master int32) (xPointer, bool) {
	if c.win == 0 || master == 0 {
		return xPointer{}, false
	}
	var root, child uint
	var rx, ry, wx, wy float64
	var buttons xiButtonState
	var mods, group xiModState
	if xiQueryPointer(c.d, master, c.win, &root, &child, &rx, &ry, &wx, &wy, &buttons, &mods, &group) == 0 {
		return xPointer{}, false
	}
	if buttons.Mask != nil {
		xFree(buttons.Mask)
	}
	inside := wx >= 0 && wy >= 0 && wx < float64(c.w) && wy < float64(c.h)
	return xPointer{X: wx, Y: wy, Inside: inside}, true
}

// goString copies a C string.
func goString(p *byte) string {
	if p == nil {
		return ""
	}
	n := 0
	for *(*byte)(unsafe.Add(unsafe.Pointer(p), n)) != 0 {
		n++
	}
	return string(unsafe.Slice(p, n))
}
