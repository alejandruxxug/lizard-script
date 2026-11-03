//go:build darwin

package terrarium

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// The macOS side of the shell: Finder, ~/.Trash and the memory from
// sysctl and vm_stat.

func hideWindow(cmd *exec.Cmd) {}

func platformReveal(path string) error { return runCommand("open", "-R", path) }

func platformOpenURL(url string) error { return runCommand("open", url) }

func platformTrashDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".Trash"), nil
}

// trashNote has nothing to write on macOS.
func trashNote(src, dest string) error { return nil }

// platformTrash asks Finder, when the plain move was not allowed.
func platformTrash(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	return runCommand("osascript", "-e", fmt.Sprintf(`tell application "Finder" to delete POSIX file %q`, abs))
}

// platformMemory reads the total from sysctl and the free memory from
// vm_stat: free and inactive pages, times the page size.
func platformMemory() (total, free uint64, err error) {
	out, err := exec.Command("sysctl", "-n", "hw.memsize").Output()
	if err != nil {
		return 0, 0, err
	}
	total, err = strconv.ParseUint(strings.TrimSpace(string(out)), 10, 64)
	if err != nil {
		return 0, 0, err
	}
	out, err = exec.Command("vm_stat").Output()
	if err != nil {
		return 0, 0, err
	}
	free, err = parseVMStat(string(out))
	return total, free, err
}

// parseVMStat turns vm_stat's text into free bytes (free + inactive pages).
func parseVMStat(text string) (uint64, error) {
	page, pages, found := uint64(0), uint64(0), 0
	for _, line := range strings.Split(text, "\n") {
		switch {
		case strings.Contains(line, "page size of"):
			for _, f := range strings.Fields(line) {
				if n, err := strconv.ParseUint(f, 10, 64); err == nil {
					page = n
				}
			}
		case strings.HasPrefix(line, "Pages free:"), strings.HasPrefix(line, "Pages inactive:"):
			_, num, _ := strings.Cut(line, ":")
			n, err := strconv.ParseUint(strings.Trim(strings.TrimSpace(num), "."), 10, 64)
			if err != nil {
				return 0, err
			}
			pages += n
			found++
		}
	}
	if page == 0 || found != 2 {
		return 0, fmt.Errorf("unexpected vm_stat output")
	}
	return pages * page, nil
}
