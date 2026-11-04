package tests

import (
	"bytes"
	"github.com/alejandruxxug/lizard-script/tools/testrunner"
	"testing"

	_ "github.com/alejandruxxug/lizard-script/terrarium/window" // the tests draw into headless windows too
)

// TestLily3D runs the tests of the standard module flower/lily3d in
// lily3d/ on both engines: programs with their .expected output, and
// renders compared with the golden PNGs in lily3d/golden.
func TestLily3D(t *testing.T) {
	for _, onVM := range []bool{false, true} {
		var out bytes.Buffer
		failed, err := testrunner.Run(testrunner.Options{Dir: "lily3d", VM: onVM, Out: &out})
		if err != nil || failed > 0 {
			t.Errorf("on the VM: %t, %d failed (%v):\n%s", onVM, failed, err, out.String())
		}
	}
}
