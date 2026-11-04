//go:build darwin

package window

import (
	"sync"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

// typeMagnify is NSEventTypeMagnify: the trackpad's pinch.
const typeMagnify = 30

var (
	pinchOnce    sync.Once
	pinchMonitor objc.ID
	pinchHandler objc.Block
)

// startPinch installs a local event monitor for magnify events, like the
// pen's monitor does for pressure. Call it once on the main thread, after
// the app runs. It reports whether the monitor was installed.
func startPinch() bool {
	pinchOnce.Do(func() {
		if _, err := purego.Dlopen("/System/Library/Frameworks/AppKit.framework/AppKit", purego.RTLD_GLOBAL|purego.RTLD_NOW); err != nil {
			return
		}
		selMagnification := objc.RegisterName("magnification")
		// The block hands the event back so Ebitengine still gets it.
		pinchHandler = objc.NewBlock(func(_ objc.Block, ev objc.ID) objc.ID {
			recordPinch(objc.Send[float64](ev, selMagnification))
			return ev
		})
		pinchMonitor = objc.ID(objc.GetClass("NSEvent")).Send(
			objc.RegisterName("addLocalMonitorForEventsMatchingMask:handler:"), uint64(1<<typeMagnify), pinchHandler)
	})
	return pinchMonitor != 0
}
