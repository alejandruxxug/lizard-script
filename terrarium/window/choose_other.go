//go:build !windows

package window

import "os/exec"

// hideConsole does nothing: only Windows shows a console window.
func hideConsole(cmd *exec.Cmd) {}
