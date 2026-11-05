package build

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/alejandruxxug/lizard-script/tools/bundle"
)

// A macOS program is an app: a folder Name.app with the executable in
// Contents/MacOS, the icon and the program's files in Contents/Resources and
// an Info.plist that says what they are. The files are not appended to the
// executable as on other systems: bytes after the end of a Mach-O file break
// its code signature (codesign --verify --strict fails), so they live in
// Resources/program.lzb, which bundle.Open reads.

// appSuffix ends the name of an app.
const appSuffix = ".app"

// writeApp makes the app o.Output from the runtime, the archive of the
// program and, when not nil, the icons. It signs the app ad hoc when
// codesign is installed (a signature that names no developer, but gives the
// app a valid seal, which Apple silicon wants to run an unsigned download).
func writeApp(o Options, base, archive []byte, icons *iconSet) error {
	name := strings.TrimSuffix(filepath.Base(o.Output), appSuffix)
	if err := os.RemoveAll(o.Output); err != nil {
		return err
	}
	macos := filepath.Join(o.Output, "Contents", "MacOS")
	resources := filepath.Join(o.Output, "Contents", "Resources")
	for _, dir := range []string{macos, resources} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	files := map[string][]byte{
		filepath.Join(macos, name):                        base,
		filepath.Join(resources, bundle.AppFile):          archive,
		filepath.Join(o.Output, "Contents", "Info.plist"): infoPlist(name, icons != nil),
	}
	if icons != nil {
		files[filepath.Join(resources, "icon.icns")] = writeICNS(icons)
	}
	for path, data := range files {
		mode := os.FileMode(0o644)
		if filepath.Dir(path) == macos {
			mode = 0o755
		}
		if err := os.WriteFile(path, data, mode); err != nil {
			return err
		}
	}
	return signApp(o.Output)
}

// signApp signs the app ad hoc. Without codesign (a build on another
// system) the app is left unsigned, and a Mac signs it on its own at its
// first run only if it is arm64 and has Go's signature, so say nothing.
func signApp(app string) error {
	codesign, err := exec.LookPath("codesign")
	if err != nil {
		return nil
	}
	if text, err := exec.Command(codesign, "--force", "--deep", "-s", "-", app).CombinedOutput(); err != nil {
		return fmt.Errorf("codesign: %v\n%s", err, text)
	}
	return nil
}

// notID matches what a bundle identifier can't hold.
var notID = regexp.MustCompile(`[^A-Za-z0-9.-]+`)

// infoPlist is the Info.plist of the app called name.
func infoPlist(name string, icon bool) []byte {
	id := "app.lizardscript." + strings.Trim(notID.ReplaceAllString(name, "-"), "-")
	iconKey := ""
	if icon {
		iconKey = "\t<key>CFBundleIconFile</key>\n\t<string>icon</string>\n"
	}
	text := name
	for _, r := range [][2]string{{"&", "&amp;"}, {"<", "&lt;"}, {">", "&gt;"}} {
		text = strings.ReplaceAll(text, r[0], r[1])
	}
	return []byte(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleName</key>
	<string>` + text + `</string>
	<key>CFBundleDisplayName</key>
	<string>` + text + `</string>
	<key>CFBundleExecutable</key>
	<string>` + text + `</string>
	<key>CFBundleIdentifier</key>
	<string>` + id + `</string>
	<key>CFBundlePackageType</key>
	<string>APPL</string>
	<key>CFBundleInfoDictionaryVersion</key>
	<string>6.0</string>
` + iconKey + `	<key>NSHighResolutionCapable</key>
	<true/>
</dict>
</plist>
`)
}
