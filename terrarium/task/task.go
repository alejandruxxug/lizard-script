// Package task is the Terrarium's task module: it runs a function of a
// module in the background, while the program goes on.
//
//	let t = task.start("trace", "trace_image", [path, options])
//	...
//	if t.done() { let shapes = t.result() }
//
// A task runs in a fresh interpreter of its own, in a process of its own:
// the program's executable started again (see worker.go), on the engine the
// program runs on. Nothing is shared: the arguments and the result are
// copied across as data (see wire.go), so a task can't change the
// program's values or see its variables, and many tasks run at the same
// time on as many processors. Cancelling a task ends its process at once.
//
// A host that imports this package gets the module in terrarium.Install,
// and must call ServeIfWorker first thing in main, so its executable can
// run tasks:
//
//	import "github.com/alejandruxxug/lizard-script/terrarium/task"
package task

import (
	"strings"

	"github.com/alejandruxxug/lizard-script/engine/evaluator"
	"github.com/alejandruxxug/lizard-script/engine/object"
	"github.com/alejandruxxug/lizard-script/lang/ast"
	"github.com/alejandruxxug/lizard-script/lang/token"
	"github.com/alejandruxxug/lizard-script/terrarium"
	"github.com/alejandruxxug/lizard-script/terrarium/internal/arg"
)

func init() {
	terrarium.Register(func(env *object.Environment) {
		env.Define("task", arg.Module("task", map[string]object.BuiltinFunction{
			"start": start,
		}))
	})
}

// task.start(module, function_name, args = []) starts a task that calls
// function_name(args...) of module, and returns its handle at once. The
// module is found the way use finds it: next to the file that calls
// task.start, then in the module folders and the standard modules (in a
// built program, among its own files). A module that isn't there is the
// error now; any other error of the task comes from t.result().
func start(args ...object.Object) object.Object {
	if err := arg.Count(args, 2, 3); err != nil {
		return err
	}
	module, err := arg.String(args, 0, "module")
	if err != nil {
		return err
	}
	function, err := arg.String(args, 1, "function name")
	if err != nil {
		return err
	}
	var list []object.Object
	if len(args) == 3 {
		a, err := arg.Array(args, 2, "list of arguments")
		if err != nil {
			return err
		}
		list = a.Elements
	}
	if served || IsWorker() {
		// Inside a task the module says it is not available; this is a
		// process that should have served a task and didn't.
		return arg.Errorf("not available inside a task")
	}

	found, ferr := findModule(module)
	if ferr != nil {
		return ferr
	}
	r := request{
		module:      found,
		function:    function,
		args:        list,
		vm:          terrarium.OnVM,
		programFile: terrarium.ProgramFile,
		programArgs: terrarium.Args,
		modulePath:  evaluator.ModulePath,
	}
	encoded, eerr := encodeRequest(r)
	if eerr != nil {
		return arg.Errorf("%v", eerr)
	}
	return handle(launch(encoded))
}

// findModule finds the module the way use does, from the file that called
// task.start, and gives its source name.
func findModule(module string) (string, *object.Error) {
	at := token.Token{Type: token.IDENT, Literal: "task.start", Source: evaluator.CallSource()}
	node := &ast.UseStatement{Token: at, Path: &ast.StringLiteral{Token: at, Value: module}}
	name, _, err := evaluator.FindModule(node)
	if err != nil {
		// The message without the position of the made-up use.
		message := err.Message
		if _, after, ok := strings.Cut(message, "runtime error: "); ok {
			message = after
		}
		return "", arg.Errorf("%s", message)
	}
	return name, nil
}

// handle is the hash a program holds a task by: its id and its functions.
func handle(t *job) *object.Hash {
	h := arg.Module("task", map[string]object.BuiltinFunction{
		"done":   t.doneFn,
		"result": t.resultFn,
		"cancel": t.cancelFn,
		"state":  t.stateFn,
	})
	arg.Set(h, "id", arg.Int64(t.id))
	return h
}

// t.done() is whether the task has ended: done, failed or cancelled.
func (t *job) doneFn(args ...object.Object) object.Object {
	if err := arg.Count(args, 0, 0); err != nil {
		return err
	}
	return arg.Bool(t.current() != stateRunning)
}

// t.state() is "running" (or waiting for a free processor), "done",
// "failed" or "cancelled".
func (t *job) stateFn(args ...object.Object) object.Object {
	if err := arg.Count(args, 0, 0); err != nil {
		return err
	}
	return arg.Str(t.current())
}

// t.cancel() stops the task: its process ends at once, and it is
// "cancelled" from now on. A task that has ended stays as it was.
func (t *job) cancelFn(args ...object.Object) object.Object {
	if err := arg.Count(args, 0, 0); err != nil {
		return err
	}
	t.cancel()
	return arg.Null
}

// t.result() waits for the task to end and gives the value its function
// returned, a copy of it, the same one each time. The error of a task that
// failed or was cancelled is raised here, at the call of t.result().
func (t *job) resultFn(args ...object.Object) object.Object {
	if err := arg.Count(args, 0, 0); err != nil {
		return err
	}
	<-t.ended
	t.mu.Lock()
	defer t.mu.Unlock()
	switch t.state {
	case stateCancelled:
		return arg.Errorf("the task was cancelled")
	case stateFailed:
		return arg.Errorf("the task failed: %s", t.failure)
	}
	if t.value == nil {
		// The value is read here, on the program's side, the first time.
		value, _, err := decode(t.answer[1:])
		if err != nil {
			return arg.Errorf("the task's answer is broken: %v", err)
		}
		t.value, t.answer = value, nil
	}
	return t.value
}

// shownError is a task's error the way the command line shows it: with the
// file it happened in, when it has one.
func shownError(e *object.Error) string {
	if e.Source == "" {
		return e.Message
	}
	return e.Source + ": " + e.Message
}
