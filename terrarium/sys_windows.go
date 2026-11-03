//go:build windows

package terrarium

import (
	"os/exec"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// The Windows side of clip and os.locale. It compiles and vets everywhere,
// but needs a real Windows PC to be tried.

// powershell runs a script with no profile and no console window, and
// returns its output.
func powershell(stdin, script string) ([]byte, error) {
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script)
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: windows.CREATE_NO_WINDOW}
	return cmd.Output()
}

const psUTF8 = "[Console]::OutputEncoding=[Text.Encoding]::UTF8;[Console]::InputEncoding=[Text.Encoding]::UTF8;"

func platformClipGet() (string, bool) {
	out, err := powershell("", psUTF8+"Get-Clipboard -Raw")
	return strings.TrimPrefix(string(out), "\xef\xbb\xbf"), err == nil
}

func platformClipSet(text string) {
	if text == "" {
		powershell("", "Set-Clipboard -Value $null")
		return
	}
	powershell(text, psUTF8+"Set-Clipboard -Value ([Console]::In.ReadToEnd())")
}

// platformLocale asks Windows for the user's locale name ("es-CO").
func platformLocale() string {
	proc := windows.NewLazySystemDLL("kernel32.dll").NewProc("GetUserDefaultLocaleName")
	var buf [85]uint16 // LOCALE_NAME_MAX_LENGTH
	n, _, _ := proc.Call(uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if n == 0 {
		return ""
	}
	return twoLetters(windows.UTF16ToString(buf[:]))
}
