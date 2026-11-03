package window

import "time"

// headlessLoop runs the frames of a window that has no screen: one after
// the other, as fast as tick goes, so a test or a benchmark doesn't wait.
// With automation on it keeps the window's fps instead, so a driver has
// time to send its commands. The window's events are the ones injected.
// An on_demand window with nothing left to do and no driver stops, since
// nothing can happen to it any more.
func headlessLoop(w *Window, frame func(events []Event) bool) error {
	for {
		start := time.Now()
		if !frame(nil) || (w.idle && w.auto == nil) {
			return nil
		}
		if w.auto != nil {
			time.Sleep(time.Second/time.Duration(w.fps) - time.Since(start))
		}
	}
}
