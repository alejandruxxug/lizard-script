//go:build windows

package terrarium

import (
	"errors"
	"os/exec"
	"path/filepath"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// The Windows side of the shell: Explorer, the Recycle Bin and
// GlobalMemoryStatusEx. It compiles and vets everywhere, but needs a real
// Windows PC to be tried.

// hideWindow keeps a console window from flashing.
func hideWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: windows.CREATE_NO_WINDOW}
}

// platformReveal opens Explorer with the file selected. explorer.exe often
// exits with status 1 even when it worked, so its exit code is ignored.
func platformReveal(path string) error {
	err := runCommand("explorer", "/select,"+filepath.FromSlash(path))
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return nil
	}
	return err
}

// platformOpenURL hands the address to the shell's default program.
func platformOpenURL(url string) error {
	return runCommand("rundll32", "url.dll,FileProtocolHandler", url)
}

// There is no folder to move into: the Recycle Bin is only reachable
// through the shell.
func platformTrashDir() (string, error) { return "", errNoTrashDir }

func trashNote(src, dest string) error { return nil }

// platformTrash sends the file to the Recycle Bin. The path goes in an
// environment variable, so nothing in it is read as PowerShell.
func platformTrash(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	script := "Add-Type -AssemblyName Microsoft.VisualBasic;" +
		"$p=$env:LIZARD_TRASH;" +
		"if (Test-Path -LiteralPath $p -PathType Container) " +
		"{[Microsoft.VisualBasic.FileIO.FileSystem]::DeleteDirectory($p,'OnlyErrorDialogs','SendToRecycleBin')} " +
		"else {[Microsoft.VisualBasic.FileIO.FileSystem]::DeleteFile($p,'OnlyErrorDialogs','SendToRecycleBin')}"
	return runCommandEnv([]string{"LIZARD_TRASH=" + abs}, "powershell", "-NoProfile", "-NonInteractive", "-Command", script)
}

// memoryStatusEx is the MEMORYSTATUSEX struct of GlobalMemoryStatusEx.
type memoryStatusEx struct {
	Length               uint32
	MemoryLoad           uint32
	TotalPhys            uint64
	AvailPhys            uint64
	TotalPageFile        uint64
	AvailPageFile        uint64
	TotalVirtual         uint64
	AvailVirtual         uint64
	AvailExtendedVirtual uint64
}

func platformMemory() (total, free uint64, err error) {
	proc := windows.NewLazySystemDLL("kernel32.dll").NewProc("GlobalMemoryStatusEx")
	st := memoryStatusEx{Length: uint32(unsafe.Sizeof(memoryStatusEx{}))}
	if r, _, e := proc.Call(uintptr(unsafe.Pointer(&st))); r == 0 {
		return 0, 0, e
	}
	return st.TotalPhys, st.AvailPhys, nil
}
