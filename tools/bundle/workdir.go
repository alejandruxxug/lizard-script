package bundle

import (
	"os"
	"path/filepath"
)

// RestoreWorkingDir undoes a move the windowing library makes on macOS: in
// an app bundle it changes the working directory to Contents/Resources while
// the program starts, as GLFW does by default. A command typed in a shell
// (`Name.app/Contents/MacOS/name render pic.lzdraw`) expects relative paths
// to mean the shell's folder, so when the working directory is a bundle's
// Resources folder and the shell said where it was (PWD), go back there.
// Started from Finder there is no shell folder, and nothing changes.
func RestoreWorkingDir() {
	wd, err := os.Getwd()
	if err != nil {
		return
	}
	if to := restoredDir(wd, os.Getenv("PWD")); to != "" {
		_ = os.Chdir(to)
	}
}

// restoredDir is the folder to go back to, or "" to stay.
func restoredDir(wd, pwd string) string {
	if pwd == "" || pwd == wd || filepath.Base(wd) != "Resources" || filepath.Base(filepath.Dir(wd)) != "Contents" {
		return ""
	}
	if info, err := os.Stat(pwd); err != nil || !info.IsDir() {
		return ""
	}
	return pwd
}
