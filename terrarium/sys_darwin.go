//go:build darwin

package terrarium

import (
	"os/exec"
	"strings"
)

// The macOS side of clip and os.locale.

func platformClipGet() (string, bool) {
	out, err := exec.Command("pbpaste").Output()
	return string(out), err == nil
}

func platformClipSet(text string) {
	cmd := exec.Command("pbcopy")
	cmd.Stdin = strings.NewReader(text)
	cmd.Run()
}

// platformLocale reads the first of the user's preferred languages.
func platformLocale() string {
	out, err := exec.Command("defaults", "read", "-g", "AppleLanguages").Output()
	if err != nil {
		return ""
	}
	// ( "es-CO", "en-US" ): the first quoted entry
	text := string(out)
	if i := strings.IndexByte(text, '"'); i >= 0 {
		return twoLetters(text[i+1:])
	}
	return ""
}
