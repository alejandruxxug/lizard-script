package window

import (
	"path/filepath"
	"strings"

	"github.com/alejandruxxug/lizard-script/engine/object"
	"github.com/alejandruxxug/lizard-script/terrarium/internal/arg"
)

// window.choose_save(w, options = {}) opens the system's "save file"
// dialog, where the user picks a folder and a file name; the system asks
// about overwriting. Like window.choose_file it returns at once, and the
// answer is a "chosen" event (dialog "save") with the one path, or an empty
// list on cancel.

// saveKeys are the options window.choose_save takes.
var saveKeys = []string{"title", "name", "types", "folder", "id"}

// chooseSave is window.choose_save(w, options = {}).
func chooseSave(args ...object.Object) object.Object {
	if err := arg.Count(args, 1, 2); err != nil {
		return err
	}
	w, err := windowArg(args, 0)
	if err != nil {
		return err
	}
	opts := chooseOptions{Save: true}
	if len(args) == 2 {
		h, err := arg.Hash(args, 1, "options")
		if err != nil {
			return err
		}
		if opts, err = chooseOptionsOf(h, saveKeys); err != nil {
			return err
		}
		opts.Save = true
	}
	return launch(w, opts)
}

// savedPath is the path a save dialog answered with: at most one. On macOS
// the dialog has no file types, so a name without an extension gets the
// first type's.
func savedPath(goos string, paths, types []string) []string {
	if len(paths) == 0 {
		return paths
	}
	path := paths[0]
	if goos == "darwin" && len(types) > 0 && filepath.Ext(path) == "" {
		path += "." + types[0]
	}
	return []string{path}
}

// macSaveDialog is osascript's "choose file name", which prints one path.
func macSaveDialog(o chooseOptions) dialogCommand {
	choose := "choose file name"
	if o.Title != "" {
		choose += " with prompt " + appleString(o.Title)
	}
	if o.Name != "" {
		choose += " default name " + appleString(o.Name)
	}
	if o.Folder != "" {
		choose += " default location (POSIX file " + appleString(o.Folder) + ")"
	}
	script := "set picked to " + choose + "\nreturn POSIX path of picked"
	return dialogCommand{Name: "osascript", Args: []string{"-e", script}}
}

// windowsSaveDialog is PowerShell's SaveFileDialog. The options go in
// environment variables, like the open dialog's.
func windowsSaveDialog(o chooseOptions) dialogCommand {
	script := strings.Join([]string{
		"Add-Type -AssemblyName System.Windows.Forms",
		"[Console]::OutputEncoding=[Text.Encoding]::UTF8",
		"$d=New-Object System.Windows.Forms.SaveFileDialog",
		"$d.OverwritePrompt=$true",
		"if ($env:LIZARD_TITLE) {$d.Title=$env:LIZARD_TITLE}",
		"if ($env:LIZARD_FOLDER) {$d.InitialDirectory=$env:LIZARD_FOLDER}",
		"if ($env:LIZARD_NAME) {$d.FileName=$env:LIZARD_NAME}",
		"if ($env:LIZARD_FILTER) {$d.Filter=$env:LIZARD_FILTER}",
		"if ($env:LIZARD_EXT) {$d.DefaultExt=$env:LIZARD_EXT; $d.AddExtension=$true}",
		"if ($d.ShowDialog() -eq 'OK') {[Console]::Out.WriteLine($d.FileName)}",
	}, ";")
	ext := ""
	if len(o.Types) > 0 {
		ext = o.Types[0]
	}
	return dialogCommand{
		Name: "powershell",
		Args: []string{"-NoProfile", "-NonInteractive", "-STA", "-Command", script},
		Env: []string{
			"LIZARD_TITLE=" + o.Title,
			"LIZARD_FOLDER=" + o.Folder,
			"LIZARD_NAME=" + o.Name,
			"LIZARD_FILTER=" + windowsFilter(o.Types),
			"LIZARD_EXT=" + ext,
		},
	}
}

// startPath is the folder and name a Linux dialog starts at: "folder/name".
func startPath(o chooseOptions) string {
	if o.Folder == "" {
		return o.Name
	}
	return strings.TrimSuffix(o.Folder, "/") + "/" + o.Name
}

// zenitySaveDialog is zenity's save selection, asking before it overwrites.
func zenitySaveDialog(o chooseOptions) dialogCommand {
	args := []string{"--file-selection", "--save", "--confirm-overwrite"}
	if o.Title != "" {
		args = append(args, "--title="+o.Title)
	}
	if start := startPath(o); start != "" {
		args = append(args, "--filename="+start)
	}
	if len(o.Types) > 0 {
		globs := make([]string, len(o.Types))
		for i, t := range o.Types {
			globs[i] = "*." + t
		}
		args = append(args, "--file-filter=Files | "+strings.Join(globs, " "))
	}
	return dialogCommand{Name: "zenity", Args: args}
}

// kdialogSaveDialog is kdialog's save-file dialog.
func kdialogSaveDialog(o chooseOptions) dialogCommand {
	start := startPath(o)
	if o.Folder == "" {
		start = "./" + o.Name
	}
	args := []string{"--getsavefilename", start}
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
	return dialogCommand{Name: "kdialog", Args: args}
}
