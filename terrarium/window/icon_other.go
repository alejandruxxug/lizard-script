//go:build !darwin

package window

// setDockIcon does nothing: there is no Dock; Ebitengine's SetWindowIcon
// shows the icon on Windows and Linux.
func setDockIcon(*windowIcon) {}
