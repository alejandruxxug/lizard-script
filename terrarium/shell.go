package terrarium

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// The desktop's shell, behind fs.trash, os.reveal, os.open_url and
// os.memory: the Trash, the file manager, the browser and the machine's
// memory. Each system does it its own way (shell_darwin.go, shell_windows.go
// and shell_other.go, for Linux and the other Unix systems).

// runCommand runs a program and waits for it. Tests replace it.
var runCommand = func(name string, args ...string) error {
	return runCommandEnv(nil, name, args...)
}

// runCommandEnv is runCommand with more environment variables.
func runCommandEnv(env []string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	if len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
	}
	hideWindow(cmd)
	return cmd.Run()
}

// trashDir is the Trash folder to move files into (the files folder of the
// freedesktop Trash on Linux). On Windows there is none: errNoTrashDir.
// Tests replace it.
var trashDir = platformTrashDir

// errNoTrashDir means there is no Trash folder to move files into: the
// system's own way (platformTrash) is used instead.
var errNoTrashDir = errors.New("no trash folder")

// Trash moves a file or folder into the Trash. It never deletes: a name
// already in the Trash gets " 2", " 3", ... before its extension. If the
// move is not allowed (another disk) it asks the system (Finder, gio, the
// Recycle Bin), and if that fails too, the file stays.
func Trash(path string) error {
	path = filepath.FromSlash(path)
	if _, err := os.Lstat(path); err != nil {
		return err
	}
	if dir, err := trashDir(); err == nil && os.MkdirAll(dir, 0o700) == nil {
		dest := freeName(dir, filepath.Base(path))
		if trashNote(path, dest) == nil && os.Rename(path, dest) == nil {
			return nil
		}
	}
	if err := platformTrash(path); err == nil {
		return nil
	}
	return errors.New("the Trash didn't take it")
}

// freeName is a path in dir for the name base that nothing has yet.
func freeName(dir, base string) string {
	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)
	dest := filepath.Join(dir, base)
	for n := 2; ; n++ {
		if _, err := os.Lstat(dest); err != nil {
			return dest
		}
		dest = filepath.Join(dir, fmt.Sprintf("%s %d%s", stem, n, ext))
	}
}

// trashInfoText is the content of the freedesktop Trash's
// info/<name>.trashinfo, which lets a file manager put a file back.
func trashInfoText(origPath string, when time.Time) string {
	return "[Trash Info]\nPath=" + percentEncodePath(origPath) +
		"\nDeletionDate=" + when.Format("2006-01-02T15:04:05") + "\n"
}

// percentEncodePath escapes a path the way the freedesktop spec wants:
// everything but letters, digits, "-_.~" and "/" becomes %XX.
func percentEncodePath(p string) string {
	const hex = "0123456789ABCDEF"
	var b strings.Builder
	for i := 0; i < len(p); i++ {
		c := p[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9',
			c == '-', c == '_', c == '.', c == '~', c == '/':
			b.WriteByte(c)
		default:
			b.WriteByte('%')
			b.WriteByte(hex[c>>4])
			b.WriteByte(hex[c&15])
		}
	}
	return b.String()
}

// Reveal shows a file in the file manager: selected in Finder or Explorer,
// its folder on Linux.
func Reveal(path string) error {
	path = filepath.FromSlash(path)
	if _, err := os.Stat(path); err != nil {
		return err
	}
	return platformReveal(path)
}

// OpenURL opens a web address (http, https or mailto) in the user's
// browser or mail program. Anything else is refused, so a program can't run
// a file through it.
func OpenURL(url string) error {
	lower := strings.ToLower(url)
	if !strings.HasPrefix(lower, "http://") && !strings.HasPrefix(lower, "https://") && !strings.HasPrefix(lower, "mailto:") {
		return errors.New("only http, https and mailto addresses can be opened")
	}
	return platformOpenURL(url)
}

// Memory is the machine's memory in bytes: all of it, and what is free to
// use now (free, and what the system can give back at once). Free is never
// more than total.
func Memory() (total, free uint64, err error) {
	total, free, err = platformMemory()
	if err != nil {
		return 0, 0, err
	}
	return total, min(free, total), nil
}
