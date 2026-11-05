//go:build !windows

package task

import "os/exec"

// hideConsole does nothing: only Windows opens a console for a process.
func hideConsole(cmd *exec.Cmd) {}
