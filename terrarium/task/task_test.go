package task

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/alejandruxxug/lizard-script/engine/evaluator"
	"github.com/alejandruxxug/lizard-script/engine/object"
	"github.com/alejandruxxug/lizard-script/terrarium"
	_ "github.com/alejandruxxug/lizard-script/terrarium/window" // so a task has a window module to refuse
	"github.com/alejandruxxug/lizard-script/tools/bundle"
	"github.com/alejandruxxug/lizard-script/tools/repl"
)

// The test binary is the workers' executable too: started as one, it runs
// the task and exits instead of running the tests. Started with
// programEnv, it runs that program and exits, as lizardscript would.
func TestMain(m *testing.M) {
	ServeIfWorker()
	if file := os.Getenv(programEnv); file != "" {
		os.Exit(runProgram(file))
	}
	os.Exit(m.Run())
}

const programEnv = "TASK_TEST_PROGRAM"

func runProgram(file string) int {
	source, err := os.ReadFile(file)
	if err != nil {
		return 2
	}
	if !repl.RunFile(file, string(source), os.Stdout, os.Stderr, false) {
		return 1
	}
	return 0
}

// work is the module the tests run their tasks from.
const work = `
let sum = function(list) {
  let total = 0
  for (let i = 0; i < len(list); i += 1) { total += list[i] }
  total
}
let touch = function(h) { h["seen"] = true; h["list"][0] = 99; h }
let boom = function() { error("boom") }
let forever = function() { while true { } }
let beat = function(file) { while true { fs.write(file, str(time.ms())); time.sleep(10) } }
let give_function = function() { {"f": function() { 1 }} }
let values = function() { [(1, 2), #ff000080, 1.5, null, true, "s", {"a": [1]}] }
let open_window = function() { window.open(10, 10) }
let read_clip = function() { clip.get() }
let start_task = function() { task.start("work", "sum", [[1]]) }
let quit = function() { os.exit(1) }
let platform = function() { os.platform() }
`

// program writes the files of a program into a new folder: main.liz with
// code and work.liz, and returns the path of main.liz.
func program(t *testing.T, code string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "work.liz"), []byte(work), 0o644); err != nil {
		t.Fatal(err)
	}
	main := filepath.Join(dir, "main.liz")
	if err := os.WriteFile(main, []byte(code), 0o644); err != nil {
		t.Fatal(err)
	}
	return filepath.ToSlash(main)
}

// runCode runs code as a program next to work.liz, on one engine, and returns
// what it printed and its error.
func runCode(t *testing.T, code string, onVM bool) string {
	t.Helper()
	main := program(t, code)
	var out bytes.Buffer
	repl.RunFile(main, code, &out, &out, onVM)
	return strings.ReplaceAll(out.String(), filepath.Dir(filepath.FromSlash(main))+string(filepath.Separator), "")
}

func TestTasks(t *testing.T) {
	tests := []struct {
		name string
		code string
		want string
	}{
		{"a result", `puts(task.start("work", "sum", [[1, 2, 3]]).result())`, "6\n"},
		{"no arguments", `puts(task.start("work", "values").result())`,
			"[(1, 2), #ff000080, 1.5, null, true, \"s\", {\"a\": [1]}]\n"},
		{"states", `let t = task.start("work", "sum", [[1]])
puts(t.state(), t.done())
t.result()
puts(t.state(), t.done(), t.id > 0)`, "running false\ndone true true\n"},
		{"the same result each time", `let t = task.start("work", "values")
puts(t.result() === t.result())`, "true\n"},
		{"a deep copy", `let h = {"list": [1, 2]}
let back = task.start("work", "touch", [h]).result()
puts(h)
puts(back)`, "{\"list\": [1, 2]}\n{\"list\": [99, 2], \"seen\": true}\n"},
		{"an error", `let t = task.start("work", "boom")
try { t.result() } catch (e) { puts(e["message"]) }
puts(t.state())`, "task.result: the task failed: work.liz: [line 8, column 30] runtime error: boom\nfailed\n"},
		{"an error is raised at the caller", `task.start("work", "boom").result()`,
			"[line 1, column 34] runtime error: task.result: the task failed: work.liz: [line 8, column 30] runtime error: boom\n"},
		{"a function refused", `task.start("work", "touch", [{"a": [1, function() { 2 }]}])`,
			"[line 1, column 11] runtime error: task.start: args[0][\"a\"][1] is a FUNCTION, and only data crosses to and from a task (numbers, strings, booleans, null, lists, hashes, points and colors)\n"},
		{"a function given back", `try { task.start("work", "give_function").result() } catch (e) { puts(e["message"]) }`,
			"task.result: the task failed: the result[\"f\"] is a FUNCTION, and only data crosses to and from a task (numbers, strings, booleans, null, lists, hashes, points and colors)\n"},
		{"window", `try { task.start("work", "open_window").result() } catch (e) { puts(e["message"]) }`,
			"task.result: the task failed: work.liz: [line 13, column 43] runtime error: window.open: not available inside a task\n"},
		{"clip", `try { task.start("work", "read_clip").result() } catch (e) { puts(e["message"]) }`,
			"task.result: the task failed: work.liz: [line 14, column 38] runtime error: clip.get: not available inside a task\n"},
		{"task", `try { task.start("work", "start_task").result() } catch (e) { puts(e["message"]) }`,
			"task.result: the task failed: work.liz: [line 15, column 41] runtime error: task.start: not available inside a task\n"},
		{"os.exit", `try { task.start("work", "quit").result() } catch (e) { puts(e["message"]) }`,
			"task.result: the task failed: work.liz: [line 16, column 32] runtime error: os.exit: not available inside a task\n"},
		{"os reads", `puts(task.start("work", "platform").result() == os.platform())`, "true\n"},
		{"no such module", `task.start("nowhere", "f")`,
			"[line 1, column 11] runtime error: task.start: can't find the module \"nowhere\": there is no nowhere.liz\n"},
		{"no such function", `try { task.start("work", "nope").result() } catch (e) { puts(e["message"]) }`,
			"task.result: the task failed: work.liz has no function nope\n"},
		{"wrong arguments", `try { task.start("work", "sum", [1, 2]).result() } catch (e) { puts(e["message"]) }`,
			"task.result: the task failed: sum(...): wrong number of arguments: want 1, got 2\n"},
		{"cancelled", `let t = task.start("work", "forever")
t.cancel()
puts(t.state(), t.done())
try { t.result() } catch (e) { puts(e["message"]) }`, "cancelled true\ntask.result: the task was cancelled\n"},
		{"cancel after the end", `let t = task.start("work", "sum", [[2]])
t.result()
t.cancel()
puts(t.state(), t.result())`, "done 2\n"},
	}
	for _, onVM := range []bool{false, true} {
		for _, tt := range tests {
			name := tt.name
			if onVM {
				name += " on the VM"
			}
			t.Run(name, func(t *testing.T) {
				if got := runCode(t, tt.code, onVM); got != tt.want {
					t.Errorf("got\n%s\nwant\n%s", got, tt.want)
				}
			})
		}
	}
}

func TestTheTaskRunsOnTheProgramsEngine(t *testing.T) {
	r := request{module: "m.liz", function: "f"}
	for _, onVM := range []bool{false, true} {
		terrarium.OnVM = onVM
		r.vm = terrarium.OnVM
		data, err := encodeRequest(r)
		if err != nil {
			t.Fatal(err)
		}
		got, err := readRequest(bufioReader(data))
		if err != nil {
			t.Fatal(err)
		}
		if got.vm != onVM || got.module != "m.liz" || got.function != "f" {
			t.Errorf("on the VM %t: read %+v", onVM, got)
		}
	}
	terrarium.OnVM = false
}

// started waits until the task's process has started.
func started(t *testing.T, j *job) {
	t.Helper()
	for deadline := time.Now().Add(10 * time.Second); time.Now().Before(deadline); time.Sleep(time.Millisecond) {
		j.mu.Lock()
		cmd := j.cmd
		j.mu.Unlock()
		if cmd != nil {
			return
		}
	}
	t.Fatal("the task's process never started")
}

// startJob starts function of work.liz in dir as a task.
func startJob(t *testing.T, dir, function string, args ...string) *job {
	t.Helper()
	r := request{module: filepath.ToSlash(filepath.Join(dir, "work.liz")), function: function}
	for _, a := range args {
		r.args = append(r.args, strObject(a))
	}
	data, err := encodeRequest(r)
	if err != nil {
		t.Fatal(err)
	}
	return launch(data)
}

func TestCancelABusyLoopAtOnce(t *testing.T) {
	dir := filepath.Dir(program(t, ""))
	j := startJob(t, dir, "forever")
	started(t, j)
	time.Sleep(100 * time.Millisecond) // it is in its loop by now
	begin := time.Now()
	j.cancel()
	<-j.ended
	if took := time.Since(begin); took > 50*time.Millisecond {
		t.Errorf("the process ended %v after cancel, want 50ms at most", took)
	}
	if j.cmd.ProcessState == nil || j.cmd.ProcessState.Success() {
		t.Errorf("the process didn't end killed: %v", j.cmd.ProcessState)
	}
}

func TestTasksWaitForAFreeProcessor(t *testing.T) {
	saved := slots
	slots = make(chan struct{}, 1)
	defer func() { slots = saved }()
	dir := filepath.Dir(program(t, ""))

	first := startJob(t, dir, "forever")
	started(t, first)
	second := startJob(t, dir, "forever")
	time.Sleep(100 * time.Millisecond)
	second.mu.Lock()
	waiting := second.cmd == nil
	second.mu.Unlock()
	if !waiting || second.current() != stateRunning {
		t.Errorf("the second task started while the first ran (state %s)", second.current())
	}
	first.cancel()
	started(t, second)
	second.cancel()
	<-second.ended

	// One that is cancelled while it waits never starts.
	first = startJob(t, dir, "forever")
	started(t, first)
	waiter := startJob(t, dir, "forever")
	waiter.cancel()
	<-waiter.ended
	first.cancel()
	<-first.ended
	if waiter.cmd != nil || waiter.current() != stateCancelled {
		t.Errorf("a task cancelled while it waited: started %t, state %s", waiter.cmd != nil, waiter.current())
	}
}

// A task that never ends doesn't keep the program alive: the program
// ends, and so does the task.
func TestATaskEndsWithTheProgram(t *testing.T) {
	beat := filepath.ToSlash(filepath.Join(t.TempDir(), "beat"))
	main := program(t, `task.start("work", "beat", ["`+beat+`"])
while !fs.exists("`+beat+`") { time.sleep(5) }
puts("bye")`)
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(self)
	cmd.Env = append(os.Environ(), programEnv+"="+main)
	done := make(chan []byte)
	go func() {
		out, _ := cmd.Output()
		done <- out
	}()
	select {
	case out := <-done:
		if string(out) != "bye\n" {
			t.Fatalf("the program printed %q", out)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("the program didn't end")
	}
	// The task wrote every 10ms; once the program is gone it stops.
	time.Sleep(300 * time.Millisecond)
	before, _ := os.ReadFile(filepath.FromSlash(beat))
	time.Sleep(300 * time.Millisecond)
	after, _ := os.ReadFile(filepath.FromSlash(beat))
	if !bytes.Equal(before, after) {
		t.Errorf("the task still runs after its program ended")
	}
}

// A built program runs its tasks from its own files, with none on disk.
func TestTasksOfABuiltProgram(t *testing.T) {
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	built := filepath.Join(t.TempDir(), "built")
	if isWindows() {
		built += ".exe"
	}
	f, err := os.OpenFile(built, os.O_CREATE|os.O_WRONLY, 0o755)
	if err != nil {
		t.Fatal(err)
	}
	in, err := os.Open(self)
	if err != nil {
		t.Fatal(err)
	}
	_, err = io.Copy(f, in)
	in.Close()
	if err != nil {
		t.Fatal(err)
	}
	main := `use "lib/work" as work
puts(task.start("lib/work", "sum", [[4, 5]]).result())`
	files := []bundle.File{{Name: "main.liz", Data: []byte(main)}, {Name: "lib/work.liz", Data: []byte(work)}}
	if err := bundle.Write(f, bundle.Program{Main: "main.liz", VM: true}, files); err != nil {
		t.Fatal(err)
	}
	f.Close()

	program, err := bundle.Open(built)
	if err != nil || program == nil {
		t.Fatalf("can't open the built program: %v", err)
	}
	savedCurrent, savedRead, savedExecutable := bundle.Current, evaluator.ReadModule, executable
	bundle.Current, evaluator.ReadModule = program, bundle.ReadFile
	executable = func() (string, error) { return built, nil }
	defer func() { bundle.Current, evaluator.ReadModule, executable = savedCurrent, savedRead, savedExecutable }()

	var out bytes.Buffer
	repl.RunFile(bundle.Path("main.liz"), main, &out, &out, true)
	if out.String() != "9\n" {
		t.Errorf("got %q, want 9", out.String())
	}
}

func bufioReader(data []byte) *bufio.Reader { return bufio.NewReader(bytes.NewReader(data)) }

func strObject(s string) object.Object { return &object.String{Value: s} }

func isWindows() bool { return runtime.GOOS == "windows" }
