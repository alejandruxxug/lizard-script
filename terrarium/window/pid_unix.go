//go:build !windows

package window

import "syscall"

// pidAlive reports whether a process with that id runs.
func pidAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, 0)
	return err == nil || err == syscall.EPERM
}
