//go:build darwin

package window

import (
	"runtime"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

// setDockIcon makes the icon the one the Dock shows for the program, with
// NSApplication's setApplicationIconImage, like the pen's monitor reaches
// AppKit through purego. A program in an app folder has its icon from the
// bundle already; this is for one run unbundled (lizardscript run). Call it
// on the main thread, after the app runs.
func setDockIcon(icon *windowIcon) {
	if _, err := purego.Dlopen("/System/Library/Frameworks/AppKit.framework/AppKit", purego.RTLD_GLOBAL|purego.RTLD_NOW); err != nil || len(icon.data) == 0 {
		return
	}
	data := objc.ID(objc.GetClass("NSData")).Send(objc.RegisterName("dataWithBytes:length:"), &icon.data[0], uint64(len(icon.data)))
	image := objc.ID(objc.GetClass("NSImage")).Send(objc.RegisterName("alloc")).Send(objc.RegisterName("initWithData:"), data)
	if image == 0 {
		return
	}
	app := objc.ID(objc.GetClass("NSApplication")).Send(objc.RegisterName("sharedApplication"))
	app.Send(objc.RegisterName("setApplicationIconImage:"), image)
	runtime.KeepAlive(icon)
}
