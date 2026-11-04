package terrarium

import (
	"github.com/alejandruxxug/lizard-script/engine/object"
	"github.com/alejandruxxug/lizard-script/terrarium/internal/arg"
	"sync"
)

// clip: the system clipboard, through the system's own tools (pbcopy on
// macOS, PowerShell on Windows, wl-copy, xclip or xsel on Linux), so no C
// compiler is needed. The last text set is also kept in the program, so
// copy and paste inside it still work where no tool is there.

var (
	clipMu    sync.Mutex
	clipLocal string
)

func clipModule() *object.Hash {
	return arg.Module("clip", map[string]object.BuiltinFunction{
		"get": clipGet,
		"set": clipSet,
	})
}

// clip.get() is the text on the clipboard ("" when there is none).
func clipGet(args ...object.Object) object.Object {
	if err := arg.Count(args, 0, 0); err != nil {
		return err
	}
	clipMu.Lock()
	defer clipMu.Unlock()
	if text, ok := platformClipGet(); ok {
		return arg.Str(text)
	}
	return arg.Str(clipLocal)
}

// clip.set(text) puts text on the clipboard.
func clipSet(args ...object.Object) object.Object {
	if err := arg.Count(args, 1, 1); err != nil {
		return err
	}
	text, err := arg.String(args, 0, "text")
	if err != nil {
		return err
	}
	clipMu.Lock()
	defer clipMu.Unlock()
	clipLocal = text
	platformClipSet(text)
	return arg.Null
}
