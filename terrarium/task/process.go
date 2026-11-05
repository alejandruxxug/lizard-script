package task

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/alejandruxxug/lizard-script/engine/object"
)

// The program's side of a task: a job starts the worker process, gives it
// the request, keeps its input open while it runs and reads its answer.
// At most one task per processor runs at a time; the others wait in line
// for a free one, and still count as running.

// The states of a task.
const (
	stateRunning   = "running"
	stateDone      = "done"
	stateFailed    = "failed"
	stateCancelled = "cancelled"
)

// slots holds a token for each task running now.
var slots = make(chan struct{}, runtime.NumCPU())

// lastID numbers the tasks of the program, from 1.
var lastID atomic.Int64

// executable is the program the workers run: this one. Tests replace it.
var executable = os.Executable

// job is one task, seen from the program.
type job struct {
	id    int64
	ended chan struct{} // closed when it ends, whatever the reason
	stop  chan struct{} // closed by cancel

	mu      sync.Mutex
	state   string
	cmd     *exec.Cmd     // its process, once it started
	answer  []byte        // what the process answered, while done
	value   object.Object // the answer read, after the first t.result()
	failure string        // why it failed
}

// launch starts a task with the encoded request.
func launch(request []byte) *job {
	t := &job{
		id:    lastID.Add(1),
		ended: make(chan struct{}),
		stop:  make(chan struct{}),
		state: stateRunning,
	}
	go t.run(request)
	return t
}

// current is the state of the task now.
func (t *job) current() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.state
}

// cancel stops a task that hasn't ended: one waiting for a processor never
// starts, and the process of one that runs is killed.
func (t *job) cancel() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.state != stateRunning {
		return
	}
	t.state = stateCancelled
	close(t.stop)
	if t.cmd != nil && t.cmd.Process != nil {
		t.cmd.Process.Kill()
	}
}

// run waits for a free processor, runs the worker and ends the task.
func (t *job) run(request []byte) {
	defer close(t.ended)
	select {
	case slots <- struct{}{}:
	case <-t.stop:
		return
	}
	defer func() { <-slots }()

	answer, err := t.work(request)
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.state == stateCancelled {
		return
	}
	if err != nil {
		t.state, t.failure = stateFailed, err.Error()
		return
	}
	if len(answer) > 0 && answer[0] == answerError {
		_, failure, err := readAnswer(answer)
		t.state = stateFailed
		if err != nil {
			t.failure = "its answer is broken: " + err.Error()
		} else {
			t.failure = shownError(failure)
		}
		return
	}
	t.state, t.answer = stateDone, answer
}

// work starts the worker, writes the request and reads the answer until
// the process ends.
func (t *job) work(request []byte) ([]byte, error) {
	self, err := executable()
	if err != nil {
		return nil, fmt.Errorf("can't find the program to run it: %v", err)
	}
	cmd := exec.Command(self)
	cmd.Env = append(os.Environ(), workerEnv+"=1")
	if _, err := os.Stderr.Stat(); err == nil {
		// What the task prints; a program with no console has none.
		cmd.Stderr = os.Stderr
	}
	hideConsole(cmd)
	in, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	cmd.Stdout = &out

	t.mu.Lock()
	if t.state == stateCancelled {
		t.mu.Unlock()
		return nil, nil
	}
	err = cmd.Start()
	if err == nil {
		t.cmd = cmd
	}
	t.mu.Unlock()
	if err != nil {
		return nil, fmt.Errorf("can't start its process: %v", err)
	}

	// The input stays open while the task runs: the worker ends when it
	// closes, which the system does when this program ends.
	if _, err := in.Write(request); err != nil && err != io.ErrClosedPipe {
		in.Close()
		cmd.Wait()
		return nil, fmt.Errorf("can't give it its work: %v", err)
	}
	err = cmd.Wait()
	in.Close()
	if out.Len() == 0 {
		if err == nil {
			err = fmt.Errorf("exit status 0")
		}
		return nil, fmt.Errorf("its process ended without an answer (%v)", err)
	}
	return out.Bytes(), nil
}
