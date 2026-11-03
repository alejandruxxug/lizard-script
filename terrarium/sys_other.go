//go:build !darwin && !windows

package terrarium

import (
	"os/exec"
	"strings"
)

// The Linux (and other Unix) side of clip and os.locale. It uses the tools
// a desktop normally has, and does without them.

// clipReaders and clipWriters are the clipboard programs to try, in order,
// with the arguments that read or write.
var clipReaders = []struct {
	name string
	args []string
}{
	{"wl-paste", []string{"--no-newline"}},
	{"xclip", []string{"-selection", "clipboard", "-o"}},
	{"xsel", []string{"--clipboard", "--output"}},
}

var clipWriters = []struct {
	name string
	args []string
}{
	{"wl-copy", nil},
	{"xclip", []string{"-selection", "clipboard", "-i"}},
	{"xsel", []string{"--clipboard", "--input"}},
}

func platformClipGet() (string, bool) {
	for _, t := range clipReaders {
		if _, err := exec.LookPath(t.name); err != nil {
			continue
		}
		if out, err := exec.Command(t.name, t.args...).Output(); err == nil {
			return string(out), true
		}
	}
	return "", false
}

func platformClipSet(text string) {
	for _, w := range clipWriters {
		if _, err := exec.LookPath(w.name); err != nil {
			continue
		}
		cmd := exec.Command(w.name, w.args...)
		cmd.Stdin = strings.NewReader(text)
		if cmd.Run() == nil {
			return
		}
	}
}

// platformLocale has nothing beyond LANG on Linux.
func platformLocale() string { return "" }
