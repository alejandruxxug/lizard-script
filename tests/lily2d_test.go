package tests

import (
	"bytes"
	"github.com/alejandruxxug/lizard-script/tools/testrunner"
	"testing"
)

// TestLily2D runs the tests of the standard module flower/lily2d in
// lily2d/ on both engines: programs with their .expected output, and
// pictures compared with the golden PNGs in lily2d/golden.
func TestLily2D(t *testing.T) {
	for _, onVM := range []bool{false, true} {
		var out bytes.Buffer
		failed, err := testrunner.Run(testrunner.Options{Dir: "lily2d", VM: onVM, Out: &out})
		if err != nil || failed > 0 {
			t.Errorf("on the VM: %t, %d failed (%v):\n%s", onVM, failed, err, out.String())
		}
	}
}
