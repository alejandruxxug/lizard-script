//go:build windows

package pen

import (
	"os"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Windows Ink (pointer messages): Wacom, Surface Pen, XP-Pen, Huion and any
// other pen the system exposes. No cgo: the Ebitengine window is subclassed
// with a callback that sees WM_POINTER* and always forwards to the original
// window procedure, so GLFW still gets everything, including the mouse
// messages Windows synthesizes from pointer input.

const (
	wmPointerEnter  = 0x0249
	wmPointerLeave  = 0x024A
	wmPointerUpdate = 0x0245
	wmPointerDown   = 0x0246
	wmPointerUp     = 0x0247

	ptPen = 3 // PT_PEN

	gwlpWndProc = ^uintptr(3) // GWLP_WNDPROC is -4
)

// pointInfo is POINTER_INFO and penInfo is POINTER_PEN_INFO.
type pointInfo struct {
	PointerType  uint32
	PointerID    uint32
	FrameID      uint32
	PointerFlags uint32
	SourceDevice uintptr
	HwndTarget   uintptr
	PixelX       int32
	PixelY       int32
	HimetricX    int32
	HimetricY    int32
	PixelRawX    int32
	PixelRawY    int32
	HimetricRawX int32
	HimetricRawY int32
	Time         uint32
	HistoryCount uint32
	InputData    int32
	KeyStates    uint32
	PerfCount    uint64
	ButtonChange int32
	_            int32
}

type penInfo struct {
	Pointer  pointInfo
	PenFlags uint32
	PenMask  uint32
	Pressure uint32
	Rotation uint32
	TiltX    int32
	TiltY    int32
}

type point struct{ X, Y int32 }

var (
	user32 = windows.NewLazySystemDLL("user32.dll")

	pGetPointerType     = user32.NewProc("GetPointerType")
	pGetPenHistory      = user32.NewProc("GetPointerPenInfoHistory")
	pSetWindowLongPtr   = user32.NewProc("SetWindowLongPtrW")
	pCallWindowProc     = user32.NewProc("CallWindowProcW")
	pEnumWindows        = user32.NewProc("EnumWindows")
	pGetWindowThreadPID = user32.NewProc("GetWindowThreadProcessId")
	pIsWindowVisible    = user32.NewProc("IsWindowVisible")
	pGetWindow          = user32.NewProc("GetWindow")
	pGetClassName       = user32.NewProc("GetClassNameW")
	pScreenToClient     = user32.NewProc("ScreenToClient")
	pGetDpiForWindow    = user32.NewProc("GetDpiForWindow") // Windows 10 1607+, optional
	kernel32            = windows.NewLazySystemDLL("kernel32.dll")
	pGetTickCount       = kernel32.NewProc("GetTickCount")

	startOnce sync.Once
	installed bool
	hook      uintptr // the callback, kept alive for the life of the process
	hwnd      uintptr
	oldProc   uintptr
)

// Start subclasses the app's window to read pen input. Call it once after the
// window exists (Ebitengine's first Update is fine). It reports whether the
// hook was installed; it is false on systems without the pointer APIs
// (before Windows 8) or when the window can't be found.
func Start() bool {
	startOnce.Do(install)
	return installed
}

// Supported reports whether Windows has the pointer calls of Windows Ink
// (Windows 8 or later).
func Supported() bool {
	for _, p := range []*windows.LazyProc{pGetPointerType, pGetPenHistory, pSetWindowLongPtr, pCallWindowProc, pEnumWindows} {
		if p.Find() != nil {
			return false
		}
	}
	return true
}

func install() {
	if !Supported() {
		return
	}
	h := findWindow()
	if h == 0 {
		return
	}
	hook = syscall.NewCallback(windowProc)
	hwnd = h
	old, _, _ := pSetWindowLongPtr.Call(h, gwlpWndProc, hook)
	if old == 0 {
		return
	}
	oldProc = old
	installed = true
}

// findWindow picks the app's window with EnumWindows, filtered by our process
// id, visible and without an owner. EnumThreadWindows needs the thread that
// created the window, but Ebitengine runs Update on its own goroutine, so the
// calling thread is not the window's thread. Among the matches, a GLFW or
// Ebitengine class wins.
func findWindow() uintptr {
	pid := uint32(os.Getpid())
	var best, any uintptr
	cb := syscall.NewCallback(func(h, _ uintptr) uintptr {
		var wpid uint32
		pGetWindowThreadPID.Call(h, uintptr(unsafe.Pointer(&wpid)))
		if wpid != pid {
			return 1
		}
		if v, _, _ := pIsWindowVisible.Call(h); v == 0 {
			return 1
		}
		if o, _, _ := pGetWindow.Call(h, 4); o != 0 { // GW_OWNER
			return 1
		}
		if any == 0 {
			any = h
		}
		var name [64]uint16
		n, _, _ := pGetClassName.Call(h, uintptr(unsafe.Pointer(&name[0])), uintptr(len(name)))
		if isGameClass(windows.UTF16ToString(name[:n])) && best == 0 {
			best = h
		}
		return 1
	})
	pEnumWindows.Call(cb, 0)
	if best != 0 {
		return best
	}
	return any
}

// windowProc sees every message of the window and forwards each one.
func windowProc(h, msg, wparam, lparam uintptr) uintptr {
	switch msg {
	case wmPointerDown, wmPointerUpdate, wmPointerUp:
		handlePointer(h, uint32(wparam&0xFFFF))
	case wmPointerLeave:
		if isPen(uint32(wparam & 0xFFFF)) {
			update(func(s *State) { *s = State{OK: true} })
		}
	}
	r, _, _ := pCallWindowProc.Call(oldProc, h, msg, wparam, lparam)
	return r
}

func isPen(id uint32) bool {
	var t uint32
	ok, _, _ := pGetPointerType.Call(uintptr(id), uintptr(unsafe.Pointer(&t)))
	return ok != 0 && t == ptPen
}

// handlePointer reads every sample the pen produced since the last message.
func handlePointer(h uintptr, id uint32) {
	if !isPen(id) {
		return
	}
	var n uint32
	if ok, _, _ := pGetPenHistory.Call(uintptr(id), uintptr(unsafe.Pointer(&n)), 0); ok == 0 || n == 0 {
		return
	}
	infos := make([]penInfo, n)
	if ok, _, _ := pGetPenHistory.Call(uintptr(id), uintptr(unsafe.Pointer(&n)), uintptr(unsafe.Pointer(&infos[0]))); ok == 0 {
		return
	}
	infos = infos[:n]
	recs := make([]penRecord, len(infos))
	for i, in := range infos {
		p := point{in.Pointer.PixelX, in.Pointer.PixelY}
		pScreenToClient.Call(h, uintptr(unsafe.Pointer(&p)))
		recs[i] = penRecord{
			PointerFlags: in.Pointer.PointerFlags, PenFlags: in.PenFlags, PenMask: in.PenMask,
			Pressure: in.Pressure, Rotation: in.Rotation, TiltX: in.TiltX, TiltY: in.TiltY,
			X: p.X, Y: p.Y, Time: in.Pointer.Time,
		}
	}
	oldestFirst(recs) // Windows lists the newest first
	dpi := windowDPI(h)
	tick, _, _ := pGetTickCount.Call()
	now := Now()
	for _, r := range recs {
		s := penSample(r, dpi, now, uint32(tick))
		eraser := penEraser(r.PenFlags)
		record(s)
		update(func(st *State) {
			st.Pressure, st.Tilt, st.Rotation = s.Pressure, s.Tilt, s.Rotation
			st.Source, st.Eraser = "tablet", eraser
		})
	}
}

// windowDPI is the window's dpi, 96 when the call is missing.
func windowDPI(h uintptr) uint32 {
	if pGetDpiForWindow.Find() != nil {
		return 96
	}
	d, _, _ := pGetDpiForWindow.Call(h)
	if d == 0 {
		return 96
	}
	return uint32(d)
}
