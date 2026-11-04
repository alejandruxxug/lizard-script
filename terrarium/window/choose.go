package window

import (
	"errors"
	"os/exec"
	"runtime"
	"strings"
	"sync"

	"github.com/alejandruxxug/lizard-script/engine/object"
	"github.com/alejandruxxug/lizard-script/terrarium/internal/arg"
)

// window.choose_file(w, options = {}) opens the system's "open file"
// dialog. It returns at once, so the window keeps running; when the user
// picks files or cancels, a "chosen" event comes with the paths (an empty
// list on cancel). The dialog is the system's own program (osascript,
// PowerShell, zenity or kdialog) run in a goroutine, so nothing here needs
// the window's thread.

// chooseOptions are the options of window.choose_file.
type chooseOptions struct {
	Title  string
	Types  []string // extensions, without the dot: "png"
	Folder string   // where the dialog starts
	Many   bool     // more than one file may be picked
}

// chooseKeys are the options window.choose_file takes.
var chooseKeys = []string{"title", "types", "folder", "many"}

// dialogCommand is a program to run, with extra environment variables.
type dialogCommand struct {
	Name string
	Args []string
	Env  []string
}

// chooser holds the events of finished dialogs until the next frame. It is
// the only part of a Window that another goroutine touches.
type chooser struct {
	mu      sync.Mutex
	open    bool    // a dialog is on the screen
	pending []Event // finished dialogs, not yet given to the program
}

// start marks a dialog as open, or says one already is.
func (c *chooser) start() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.open {
		return false
	}
	c.open = true
	return true
}

// finish adds the "chosen" event of a dialog that ended.
func (c *chooser) finish(paths []string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.open = false
	c.pending = append(c.pending, Event{Type: "chosen", Paths: paths})
}

// take returns the events of the dialogs that ended since the last frame.
func (c *chooser) take() []Event {
	c.mu.Lock()
	defer c.mu.Unlock()
	events := c.pending
	c.pending = nil
	return events
}

// runDialog runs a dialog program and returns what it printed. Tests
// replace it, so none opens a dialog.
var runDialog = func(c dialogCommand) (string, error) {
	cmd := exec.Command(c.Name, c.Args...)
	if len(c.Env) > 0 {
		cmd.Env = append(cmd.Environ(), c.Env...)
	}
	hideConsole(cmd)
	out, err := cmd.Output()
	return string(out), err
}

// hasTool reports whether a program is installed. Tests replace it.
var hasTool = func(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// chooseFile is window.choose_file(w, options = {}).
func chooseFile(args ...object.Object) object.Object {
	if err := arg.Count(args, 1, 2); err != nil {
		return err
	}
	w, err := windowArg(args, 0)
	if err != nil {
		return err
	}
	var opts chooseOptions
	if len(args) == 2 {
		h, err := arg.Hash(args, 1, "options")
		if err != nil {
			return err
		}
		if opts, err = chooseOptionsOf(h); err != nil {
			return err
		}
	}
	cmd, cerr := dialogFor(runtime.GOOS, opts, hasTool)
	if cerr != nil {
		return arg.Errorf("%v", cerr)
	}
	if !w.choose.start() {
		return arg.Errorf("a file dialog is already open")
	}
	go func() {
		out, err := runDialog(cmd)
		if err != nil {
			out = "" // cancelled, or the program failed: nothing was chosen
		}
		w.choose.finish(parsePaths(out))
	}()
	return arg.Null
}

// chooseOptionsOf reads the options hash.
func chooseOptionsOf(h *object.Hash) (chooseOptions, *object.Error) {
	var o chooseOptions
	for _, key := range h.Order {
		name := h.Pairs[key].Key.(*object.String).Value
		if !contains(chooseKeys, name) {
			return o, unknownOption(name, chooseKeys)
		}
	}
	r := reader{h: h}
	o.Title = r.str("title", "")
	o.Folder = r.str("folder", "")
	o.Many = r.flag("many", false)
	if v := r.get("types"); v != nil {
		list, ok := v.(*object.Array)
		if !ok {
			return o, arg.Errorf("types must be a list of extensions, got %s", v.Inspect())
		}
		for _, el := range list.Elements {
			s, ok := el.(*object.String)
			if !ok {
				return o, arg.Errorf("types must be a list of extensions, got %s", el.Inspect())
			}
			if ext := strings.TrimPrefix(s.Value, "."); ext != "" {
				o.Types = append(o.Types, ext)
			}
		}
	}
	return o, nil
}

// dialogFor is the program that shows the dialog on the given system. have
// says which programs are installed (zenity or kdialog on Linux).
func dialogFor(goos string, o chooseOptions, have func(string) bool) (dialogCommand, error) {
	switch goos {
	case "darwin":
		return macDialog(o), nil
	case "windows":
		return windowsDialog(o), nil
	}
	switch {
	case have("zenity"):
		return zenityDialog(o), nil
	case have("kdialog"):
		return kdialogDialog(o), nil
	}
	return dialogCommand{}, errors.New("window.choose_file needs zenity or kdialog installed to show a file dialog")
}

// appleString is s as an AppleScript string literal.
func appleString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
}

// macDialog is osascript's "choose file", which prints one path per line.
func macDialog(o chooseOptions) dialogCommand {
	choose := "choose file"
	if o.Title != "" {
		choose += " with prompt " + appleString(o.Title)
	}
	if len(o.Types) > 0 {
		quoted := make([]string, len(o.Types))
		for i, t := range o.Types {
			quoted[i] = appleString(t)
		}
		choose += " of type {" + strings.Join(quoted, ", ") + "}"
	}
	if o.Folder != "" {
		choose += " default location (POSIX file " + appleString(o.Folder) + ")"
	}
	if o.Many {
		choose += " with multiple selections allowed"
	}
	script := []string{
		"set picked to " + choose,
		"if class of picked is not list then set picked to {picked}",
		`set out to ""`,
		"repeat with f in picked",
		"set out to out & POSIX path of f & linefeed",
		"end repeat",
		"return out",
	}
	return dialogCommand{Name: "osascript", Args: []string{"-e", strings.Join(script, "\n")}}
}

// windowsDialog is PowerShell's OpenFileDialog. The options go in
// environment variables, so nothing in them is read as PowerShell.
func windowsDialog(o chooseOptions) dialogCommand {
	script := strings.Join([]string{
		"Add-Type -AssemblyName System.Windows.Forms",
		"[Console]::OutputEncoding=[Text.Encoding]::UTF8",
		"$d=New-Object System.Windows.Forms.OpenFileDialog",
		"if ($env:LIZARD_TITLE) {$d.Title=$env:LIZARD_TITLE}",
		"if ($env:LIZARD_FOLDER) {$d.InitialDirectory=$env:LIZARD_FOLDER}",
		"if ($env:LIZARD_FILTER) {$d.Filter=$env:LIZARD_FILTER}",
		"$d.Multiselect=($env:LIZARD_MANY -eq '1')",
		"if ($d.ShowDialog() -eq 'OK') {$d.FileNames | ForEach-Object {[Console]::Out.WriteLine($_)}}",
	}, ";")
	many := "0"
	if o.Many {
		many = "1"
	}
	return dialogCommand{
		Name: "powershell",
		Args: []string{"-NoProfile", "-NonInteractive", "-STA", "-Command", script},
		Env: []string{
			"LIZARD_TITLE=" + o.Title,
			"LIZARD_FOLDER=" + o.Folder,
			"LIZARD_FILTER=" + windowsFilter(o.Types),
			"LIZARD_MANY=" + many,
		},
	}
}

// windowsFilter is the Filter text of OpenFileDialog for the extensions:
// "Files|*.png;*.jpg|All files|*.*". Nothing for no extensions.
func windowsFilter(types []string) string {
	if len(types) == 0 {
		return ""
	}
	globs := make([]string, len(types))
	for i, t := range types {
		globs[i] = "*." + t
	}
	list := strings.Join(globs, ";")
	return "Files (" + list + ")|" + list + "|All files|*.*"
}

// zenityDialog is zenity's file selection, with the paths one per line.
func zenityDialog(o chooseOptions) dialogCommand {
	args := []string{"--file-selection"}
	if o.Title != "" {
		args = append(args, "--title="+o.Title)
	}
	if o.Folder != "" {
		args = append(args, "--filename="+strings.TrimSuffix(o.Folder, "/")+"/")
	}
	if len(o.Types) > 0 {
		globs := make([]string, len(o.Types))
		for i, t := range o.Types {
			globs[i] = "*." + t
		}
		args = append(args, "--file-filter=Files | "+strings.Join(globs, " "))
	}
	if o.Many {
		args = append(args, "--multiple", "--separator=\n")
	}
	return dialogCommand{Name: "zenity", Args: args}
}

// kdialogDialog is kdialog's open-file dialog, with the paths one per line.
func kdialogDialog(o chooseOptions) dialogCommand {
	start := o.Folder
	if start == "" {
		start = "."
	}
	args := []string{"--getopenfilename", start}
	if len(o.Types) > 0 {
		globs := make([]string, len(o.Types))
		for i, t := range o.Types {
			globs[i] = "*." + t
		}
		args = append(args, strings.Join(globs, " "))
	}
	if o.Title != "" {
		args = append(args, "--title", o.Title)
	}
	if o.Many {
		args = append(args, "--multiple", "--separate-output")
	}
	return dialogCommand{Name: "kdialog", Args: args}
}

// parsePaths reads the output of a dialog: one path per line, with no
// empty lines. A cancelled dialog prints nothing, so it gives no paths.
// The list is never nil, so the program sees [] and not null.
func parsePaths(out string) []string {
	paths := []string{}
	for _, line := range strings.Split(out, "\n") {
		if line = strings.TrimSuffix(line, "\r"); line != "" {
			paths = append(paths, line)
		}
	}
	return paths
}
