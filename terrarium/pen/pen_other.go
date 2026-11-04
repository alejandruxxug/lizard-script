//go:build !darwin && !windows && !linux

package pen

// Start does nothing: only macOS, Windows and Linux report pressure.
func Start() bool { return false }

// Supported is false: this system reports no pressure.
func Supported() bool { return false }
