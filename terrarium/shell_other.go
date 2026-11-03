//go:build !darwin && !windows

package terrarium

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// The Linux (and other Unix) side of the shell: xdg-open, the freedesktop
// Trash and /proc/meminfo.

func hideWindow(cmd *exec.Cmd) {}

// platformReveal opens the file's folder: file managers have no common way
// to select a file.
func platformReveal(path string) error { return runCommand("xdg-open", filepath.Dir(path)) }

func platformOpenURL(url string) error { return runCommand("xdg-open", url) }

// platformTrashDir is the freedesktop Trash, $XDG_DATA_HOME/Trash/files.
func platformTrashDir() (string, error) {
	if x := os.Getenv("XDG_DATA_HOME"); x != "" && filepath.IsAbs(x) {
		return filepath.Join(x, "Trash", "files"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share", "Trash", "files"), nil
}

// trashNote writes the .trashinfo of a file about to be moved to dest, so
// file managers can put it back.
func trashNote(src, dest string) error {
	abs, err := filepath.Abs(src)
	if err != nil {
		return err
	}
	info := filepath.Join(filepath.Dir(filepath.Dir(dest)), "info")
	if err := os.MkdirAll(info, 0o700); err != nil {
		return err
	}
	text := trashInfoText(abs, time.Now())
	return os.WriteFile(filepath.Join(info, filepath.Base(dest)+".trashinfo"), []byte(text), 0o600)
}

// platformTrash asks gio, when the plain move was not allowed (another
// disk).
func platformTrash(path string) error {
	if _, err := exec.LookPath("gio"); err != nil {
		return errors.New("gio not found")
	}
	return runCommand("gio", "trash", path)
}

// platformMemory reads /proc/meminfo: MemTotal and MemAvailable.
func platformMemory() (total, free uint64, err error) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, 0, err
	}
	return parseMeminfo(string(data))
}

// parseMeminfo reads MemTotal and MemAvailable, in kB, as bytes.
func parseMeminfo(text string) (total, free uint64, err error) {
	found := 0
	for _, line := range strings.Split(text, "\n") {
		name, rest, ok := strings.Cut(line, ":")
		if !ok || (name != "MemTotal" && name != "MemAvailable") {
			continue
		}
		fields := strings.Fields(rest)
		if len(fields) == 0 {
			continue
		}
		kb, perr := strconv.ParseUint(fields[0], 10, 64)
		if perr != nil {
			return 0, 0, perr
		}
		if name == "MemTotal" {
			total = kb * 1024
		} else {
			free = kb * 1024
		}
		found++
	}
	if found != 2 {
		return 0, 0, fmt.Errorf("unexpected /proc/meminfo")
	}
	return total, free, nil
}
