//go:build !darwin && !windows

package pen

// Start does nothing: only macOS and Windows report pressure.
func Start() bool { return false }

// Supported is false: this system reports no pressure.
func Supported() bool { return false }
