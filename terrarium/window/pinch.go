package window

import "sync"

// The native side of the pinch: the trackpad's magnify gesture. Only macOS
// reports it (pinch_darwin.go); the other systems send no pinch events yet.
// Gestures are added up here between two frames and read once a frame.

var pinches struct {
	sync.Mutex
	scale float64 // the factor of the gestures since the last frame; 0 when none
}

// recordPinch adds a magnify gesture of magnification m (0.02 is 2% bigger,
// as AppKit reports it).
func recordPinch(m float64) {
	pinches.Lock()
	defer pinches.Unlock()
	if pinches.scale == 0 {
		pinches.scale = 1
	}
	pinches.scale *= 1 + m
}

// drainPinch gives the factor of the pinch since the last call, and false
// when there was none.
func drainPinch() (float64, bool) {
	pinches.Lock()
	defer pinches.Unlock()
	s := pinches.scale
	pinches.scale = 0
	return s, s != 0
}
