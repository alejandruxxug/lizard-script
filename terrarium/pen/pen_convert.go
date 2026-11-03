package pen

// Pure conversions for Windows Ink pen input. They live here, without a build
// tag, so every OS can test them; pen_windows.go feeds them the raw
// POINTER_PEN_INFO values.

// Windows pointer flags (POINTER_FLAG_*).
const (
	flagInContact = 0x00000004
	flagDown      = 0x00010000
	flagUp        = 0x00040000
)

// Windows pen flags (PEN_FLAG_*) and pen masks (PEN_MASK_*).
const (
	penFlagInverted = 0x00000002
	penFlagEraser   = 0x00000004

	penMaskPressure = 0x00000001
	penMaskRotation = 0x00000002
	penMaskTiltX    = 0x00000004
	penMaskTiltY    = 0x00000008
)

// penRecord is one POINTER_PEN_INFO reduced to what we use. X and Y are client
// pixels (after ScreenToClient).
type penRecord struct {
	PointerFlags uint32
	PenFlags     uint32
	PenMask      uint32
	Pressure     uint32 // 0 to 1024
	Rotation     uint32 // degrees
	TiltX, TiltY int32  // degrees, -90 to 90
	X, Y         int32
	Time         uint32 // milliseconds, the GetTickCount clock
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// penPressure scales Windows' 0..1024 to 0..1. A pen that reports no pressure
// counts as full pressure while touching and none otherwise.
func penPressure(r penRecord) float64 {
	if r.PenMask&penMaskPressure == 0 {
		if r.PointerFlags&flagInContact != 0 {
			return 1
		}
		return 0
	}
	return clamp01(float64(r.Pressure) / 1024)
}

// penTilt maps degrees (-90..90) to -1..1 on one axis.
func penTilt(deg int32) float64 { return clamp01((float64(deg)+90)/180)*2 - 1 }

// penRotation is the barrel rotation in degrees, 0 when not reported.
func penRotation(r penRecord) float64 {
	if r.PenMask&penMaskRotation == 0 {
		return 0
	}
	return float64(r.Rotation % 360)
}

// penEraser tells whether the eraser end is in use.
func penEraser(flags uint32) bool { return flags&(penFlagInverted|penFlagEraser) != 0 }

// penPhase names the gesture part a pointer sample belongs to.
func penPhase(flags uint32) string {
	switch {
	case flags&flagUp != 0:
		return PhaseUp
	case flags&flagDown != 0:
		return PhaseDown
	case flags&flagInContact != 0:
		return PhaseDrag
	}
	return PhasePen
}

// logicalPoint converts client pixels to the window's logical points. Windows
// calls 96 dpi a scale of 1, and so does Ebitengine's device scale factor.
func logicalPoint(px, py int32, dpi uint32) (float64, float64) {
	if dpi == 0 {
		dpi = 96
	}
	k := 96 / float64(dpi)
	return float64(px) * k, float64(py) * k
}

// sampleTime places a pointer timestamp on the app clock. now is Now() and
// tickNow the 32-bit tick count read at the same moment; the signed difference
// survives the tick counter wrapping.
func sampleTime(now float64, tickNow, t uint32) float64 {
	return now + float64(int32(t-tickNow))/1000
}

// penSample builds a native sample (and the state it implies) from a record.
func penSample(r penRecord, dpi uint32, now float64, tickNow uint32) Sample {
	x, y := logicalPoint(r.X, r.Y, dpi)
	phase := penPhase(r.PointerFlags)
	p := penPressure(r)
	if phase == PhaseUp || phase == PhasePen {
		p = 0
	}
	return Sample{
		T: sampleTime(now, tickNow, r.Time), X: x, Y: y, Pressure: p,
		Tilt:     [2]float64{penTilt(tiltOf(r.PenMask, penMaskTiltX, r.TiltX)), penTilt(tiltOf(r.PenMask, penMaskTiltY, r.TiltY))},
		Rotation: penRotation(r), Source: "tablet", Phase: phase,
	}
}

// tiltOf is the tilt in degrees, 0 when the pen doesn't report that axis.
func tiltOf(mask, bit uint32, deg int32) int32 {
	if mask&bit == 0 {
		return 0
	}
	return deg
}

// oldestFirst reverses Windows' newest-first pointer history in place.
func oldestFirst(rs []penRecord) {
	for i, j := 0, len(rs)-1; i < j; i, j = i+1, j-1 {
		rs[i], rs[j] = rs[j], rs[i]
	}
}

// isGameClass tells the window classes GLFW and Ebitengine register.
func isGameClass(name string) bool {
	return name == "GLFW30" || name == "GLFW31" || name == "Ebitengine" || name == "EbitengineWindowClass"
}
