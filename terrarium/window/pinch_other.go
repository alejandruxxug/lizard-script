//go:build !darwin

package window

// startPinch does nothing: only macOS reports the trackpad's pinch for now.
func startPinch() bool { return false }
