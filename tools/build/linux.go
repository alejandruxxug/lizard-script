package build

import (
	"os"
	"path/filepath"
	"strings"
)

// writeLinuxIcon puts the icon next to the program: icon.png (512 pixels)
// and icon-256.png, and name.desktop, a template of the launcher entry that
// makes a desktop show the icon. A Linux executable can't carry an icon
// itself, so the files go with it.
func writeLinuxIcon(output string, icons *iconSet) error {
	dir, name := filepath.Split(output)
	for file, size := range map[string]int{"icon.png": 512, "icon-256.png": 256} {
		if !icons.has(size) {
			continue
		}
		if err := os.WriteFile(filepath.Join(dir, file), icons.png(size), 0o644); err != nil {
			return err
		}
	}
	return os.WriteFile(filepath.Join(dir, name+".desktop"), desktopEntry(name), 0o644)
}

// desktopEntry is the launcher template of the program called name. Exec
// and Icon are names, not paths: put the program on the PATH and icon.png in
// ~/.local/share/icons/hicolor/512x512/apps/<name>.png, or write full paths.
func desktopEntry(name string) []byte {
	return []byte(strings.Join([]string{
		"[Desktop Entry]",
		"Type=Application",
		"Name=" + name,
		"Exec=" + name,
		"Icon=" + name,
		"Terminal=false",
		"Categories=Utility;",
		"",
	}, "\n"))
}
