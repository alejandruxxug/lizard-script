//go:build linux && (amd64 || arm64)

package pen

import (
	"testing"
	"unsafe"
)

// TestXLayouts checks the Go copies of the Xlib and XInput2 structs against
// the offsets of the C headers on 64-bit Linux.
func TestXLayouts(t *testing.T) {
	var cookie xCookie
	var raw xiRawEvent
	var info xiDeviceInfo
	var class xiValuatorClass
	var we xWindowEvent
	var mask xiEventMask
	for _, c := range []struct {
		name      string
		got, want uintptr
	}{
		{"XEvent size", unsafe.Sizeof(xEvent{}), 192},
		{"cookie.extension", unsafe.Offsetof(cookie.Extension), 32},
		{"cookie.evtype", unsafe.Offsetof(cookie.Evtype), 36},
		{"cookie.data", unsafe.Offsetof(cookie.Data), 48},
		{"raw.time", unsafe.Offsetof(raw.Time), 40},
		{"raw.deviceid", unsafe.Offsetof(raw.Deviceid), 48},
		{"raw.detail", unsafe.Offsetof(raw.Detail), 56},
		{"raw.valuators.mask_len", unsafe.Offsetof(raw.MaskLen), 64},
		{"raw.valuators.mask", unsafe.Offsetof(raw.Mask), 72},
		{"raw.valuators.values", unsafe.Offsetof(raw.Values), 80},
		{"raw.raw_values", unsafe.Offsetof(raw.RawValues), 88},
		{"info.name", unsafe.Offsetof(info.Name), 8},
		{"info.use", unsafe.Offsetof(info.Use), 16},
		{"info.attachment", unsafe.Offsetof(info.Attachment), 20},
		{"info.num_classes", unsafe.Offsetof(info.NumClasses), 28},
		{"info.classes", unsafe.Offsetof(info.Classes), 32},
		{"info size", unsafe.Sizeof(info), 40},
		{"class.number", unsafe.Offsetof(class.Number), 8},
		{"class.label", unsafe.Offsetof(class.Label), 16},
		{"class.min", unsafe.Offsetof(class.Min), 24},
		{"class.max", unsafe.Offsetof(class.Max), 32},
		{"configure.window", unsafe.Offsetof(we.Window), 40},
		{"configure.width", unsafe.Offsetof(we.W), 56},
		{"eventmask.mask", unsafe.Offsetof(mask.Mask), 8},
	} {
		if c.got != c.want {
			t.Errorf("%s at %d, want %d", c.name, c.got, c.want)
		}
	}
}
