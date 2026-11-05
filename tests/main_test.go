package tests

import (
	"github.com/alejandruxxug/lizard-script/terrarium/task"
	"os"
	"testing"
)

// The test binary also runs the tasks the programs start (see package
// task): started as a worker, it runs one and exits.
func TestMain(m *testing.M) {
	task.ServeIfWorker()
	os.Exit(m.Run())
}
