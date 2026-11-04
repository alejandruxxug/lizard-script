package terrarium

import (
	"embed"
	"github.com/alejandruxxug/lizard-script/engine/evaluator"
	"io/fs"
)

// The standard modules: libraries written in lizard-script that are part of
// the binary but not installed into every program, because they are big. A
// program asks for one with use, which finds it after the program's own
// folder and $LIZARD_PATH: use "lily2d" runs std/lily2d.liz, and that file
// uses its own parts in std/lily2d/. Their source names start with "std:",
// like "std:/lily2d/raster.liz".
//
//go:embed std
var standardFiles embed.FS

func init() {
	files, err := fs.Sub(standardFiles, "std")
	if err != nil {
		panic(err)
	}
	evaluator.StandardModules = files
}
