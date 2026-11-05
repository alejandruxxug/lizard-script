package task

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/alejandruxxug/lizard-script/engine/evaluator"
	"github.com/alejandruxxug/lizard-script/engine/object"
	"github.com/alejandruxxug/lizard-script/engine/vm"
	"github.com/alejandruxxug/lizard-script/lang/ast"
	"github.com/alejandruxxug/lizard-script/lang/token"
	"github.com/alejandruxxug/lizard-script/terrarium"
	"github.com/alejandruxxug/lizard-script/terrarium/internal/arg"
	"github.com/alejandruxxug/lizard-script/tools/bundle"
)

// The worker: the process a task runs in. It is the program's own
// executable started again with workerEnv set, so a built program runs its
// tasks with its own files. It reads one request on its standard input,
// runs the function, writes the answer on its standard output and ends.
// It also ends when its standard input closes: the program that started it
// cancelled the task, or ended.

// workerEnv marks a process started to run a task.
const workerEnv = "LIZARD_TASK_WORKER"

// The answer starts with one of these.
const (
	answerValue byte = iota
	answerError
)

// printPrefix starts each line a task prints.
const printPrefix = "[task] "

// unavailable are the modules a task can't use, because they need the
// program's screen and user, or would start tasks of their own.
var unavailable = []string{"window", "pen", "clip", "task"}

// unavailableOS are the functions of os a task can't use: they would end
// the task's process or open something on the user's screen. The rest of
// os only reads.
var unavailableOS = []string{"exit", "reveal", "open_url"}

// served is set once this process serves a task, so its own task.start
// knows it is inside one.
var served bool

// IsWorker reports whether this process was started to run a task.
func IsWorker() bool {
	return os.Getenv(workerEnv) != ""
}

// ServeIfWorker runs the task this process was started for and exits, when
// it was started for one; else it returns at once. A host that has the
// task module calls it first thing in main, before it looks at its
// arguments, and so does a test binary in its TestMain:
//
//	func main() {
//		task.ServeIfWorker()
//		...
//	}
func ServeIfWorker() {
	if IsWorker() {
		os.Exit(Serve(os.Stdin, os.Stdout))
	}
}

// Serve runs the task asked for on in and writes its answer to out. It
// returns the exit code of the process: 0 when the answer was written,
// even when it is the task's error.
func Serve(in io.Reader, out io.Writer) int {
	served = true
	os.Unsetenv(workerEnv) // the programs a task runs are not tasks
	// Anything that writes to the standard output, the answer's way back,
	// goes to the standard error instead.
	os.Stdout = os.Stderr

	reader := bufio.NewReader(in)
	request, err := readRequest(reader)
	if err != nil {
		fmt.Fprintln(os.Stderr, "task: can't read the request:", err)
		return 2
	}
	go func() {
		// The program closes the input to stop the task, and so does the
		// system when the program ends.
		io.Copy(io.Discard, reader)
		os.Exit(3)
	}()

	result := run(request)
	answer, err := encodeAnswer(result)
	if err != nil {
		fmt.Fprintln(os.Stderr, "task: can't write the answer:", err)
		return 2
	}
	if _, err := out.Write(answer); err != nil {
		return 2
	}
	return 0
}

// request is what the program asks a worker to do.
type request struct {
	module      string // the module's source name, as use found it
	function    string // the function of the module to call
	args        []object.Object
	vm          bool     // run on the bytecode VM
	programFile string   // terrarium.ProgramFile of the program
	programArgs []string // terrarium.Args of the program
	modulePath  []string // evaluator.ModulePath of the program
}

// encodeRequest writes r for readRequest: its length, a hash of what it
// asks and the list of the arguments. Arguments that are not data are the
// error that says where, like "args[1]["draw"] is a FUNCTION".
func encodeRequest(r request) ([]byte, error) {
	h := object.NewHash()
	arg.Set(h, "module", arg.Str(r.module))
	arg.Set(h, "function", arg.Str(r.function))
	arg.Set(h, "vm", arg.Bool(r.vm))
	arg.Set(h, "program_file", arg.Str(r.programFile))
	arg.Set(h, "program_args", stringList(r.programArgs))
	arg.Set(h, "module_path", stringList(r.modulePath))
	body, err := encode(nil, h, "the request")
	if err != nil {
		return nil, err
	}
	if body, err = encode(body, &object.Array{Elements: r.args}, "args"); err != nil {
		return nil, err
	}
	return append(binary.AppendUvarint(nil, uint64(len(body))), body...), nil
}

// readRequest reads what encodeRequest wrote.
func readRequest(in *bufio.Reader) (request, error) {
	var r request
	n, err := binary.ReadUvarint(in)
	if err != nil {
		return r, err
	}
	body := make([]byte, n)
	if _, err := io.ReadFull(in, body); err != nil {
		return r, err
	}
	header, rest, err := decode(body)
	if err != nil {
		return r, err
	}
	args, _, err := decode(rest)
	if err != nil {
		return r, err
	}
	h, ok := header.(*object.Hash)
	list, isList := args.(*object.Array)
	if !ok || !isList {
		return r, errCorrupt
	}
	get := func(key string) object.Object {
		v, _ := h.Get(&object.String{Value: key})
		return v
	}
	r.module = stringOf(get("module"))
	r.function = stringOf(get("function"))
	r.args = list.Elements
	r.vm = get("vm") == arg.True
	r.programFile = stringOf(get("program_file"))
	r.programArgs = stringsOf(get("program_args"))
	r.modulePath = stringsOf(get("module_path"))
	return r, nil
}

// run sets this process up as the program was, with the Terrarium a task
// has, and calls the function.
func run(r request) object.Object {
	setUp(r)
	env := object.NewEnvironment()
	installTask(env)

	// use "<module>" as m, from a file next to the module, so use finds
	// exactly the file the program's use found.
	at := token.Token{Type: token.IDENT, Literal: "use", Source: r.module}
	node := &ast.UseStatement{
		Token: at,
		Path:  &ast.StringLiteral{Token: at, Value: path.Base(r.module)},
		Name:  &ast.Identifier{Token: at, Value: "module"},
	}
	var runner evaluator.Runner = func(program *ast.Program, env *object.Environment) object.Object {
		return evaluator.Eval(program, env)
	}
	if r.vm {
		runner = vm.RunModule
	}
	if err := evaluator.Use(node, env, runner); isError(err) {
		return err
	}
	names, _ := env.Get("module")
	function, ok := names.(*object.Hash).Get(&object.String{Value: r.function})
	if !ok {
		return arg.Errorf("%s has no function %s", r.module, r.function)
	}
	if !isFunction(function) {
		return arg.Errorf("%s of %s is %s, not a function", r.function, r.module, named(function))
	}
	call := token.Token{Type: token.IDENT, Literal: r.function, Source: callSource}
	result := evaluator.ApplyFunction(call, function, r.args)
	if err, ok := result.(*object.Error); ok && err.Source == callSource {
		// A mistake in the call itself, like the number of arguments.
		_, message, _ := strings.Cut(err.Message, "runtime error: ")
		return arg.Errorf("%s(...): %s", r.function, message)
	}
	return result
}

// callSource is the source name of the call of the task's function, which
// is in no file.
const callSource = "task:"

// setUp makes this process see what the program sees: its files (its own
// built files too), its module folders, its main file and arguments, and
// the engine it runs on. What the task prints goes to the standard error,
// each line marked as the task's, and it reads no input.
func setUp(r request) {
	evaluator.ReadModule = bundle.ReadFile
	if self, err := os.Executable(); err == nil {
		if program, err := bundle.Open(self); err == nil {
			bundle.Current = program
		}
	}
	evaluator.ModulePath = r.modulePath
	terrarium.ProgramFile = r.programFile
	terrarium.Args = r.programArgs
	terrarium.OnVM = r.vm
	printer := &prefixWriter{out: os.Stderr, prefix: printPrefix, start: true}
	evaluator.Output = printer
	terrarium.Stderr = printer
	terrarium.Stdin = strings.NewReader("")
	evaluator.ModulePrelude = installTask
	evaluator.StartRun()
}

// installTask defines the Terrarium of a task in env: all of it, but the
// modules that need the program's screen (window, pen, clip) and task
// itself are there only to say they can't be used, and so are the
// functions of os that act.
func installTask(env *object.Environment) {
	terrarium.Install(env)
	for _, name := range unavailable {
		module, ok := env.Get(name)
		if !ok {
			continue
		}
		if h, ok := module.(*object.Hash); ok {
			env.Define(name, unavailableModule(name, h))
		}
	}
	if module, ok := env.Get("os"); ok {
		if h, ok := module.(*object.Hash); ok {
			readOnly := h.Copy()
			for _, function := range unavailableOS {
				arg.Set(readOnly, function, unavailableFunction("os."+function))
			}
			env.Define("os", readOnly)
		}
	}
}

// unavailableModule has the names of module, each a function that is the
// error of using it inside a task.
func unavailableModule(name string, module *object.Hash) *object.Hash {
	stub := object.NewHash()
	module.Each(func(pair object.HashPair) {
		stub.Set(pair.Key.(object.Hashable), unavailableFunction(name+"."+pair.Key.Inspect()))
	})
	return stub
}

func unavailableFunction(name string) *object.Builtin {
	return &object.Builtin{Name: name, Fn: func(args ...object.Object) object.Object {
		return arg.Errorf("not available inside a task")
	}}
}

// encodeAnswer writes the value of a task, or its error, for readAnswer. A
// value that is not data becomes the error that says where it is.
func encodeAnswer(result object.Object) ([]byte, error) {
	if err, ok := result.(*object.Error); ok {
		return encodeError(err)
	}
	answer, err := encode([]byte{answerValue}, result, "the result")
	if err != nil {
		return encodeError(arg.Errorf("%v", err))
	}
	return answer, nil
}

// encodeError writes the error of a task: its message and where it was.
func encodeError(e *object.Error) ([]byte, error) {
	h := object.NewHash()
	arg.Set(h, "message", arg.Str(e.Message))
	arg.Set(h, "line", arg.Int64(int64(e.Line)))
	arg.Set(h, "column", arg.Int64(int64(e.Column)))
	arg.Set(h, "source", arg.Str(e.Source))
	return encode([]byte{answerError}, h, "the error")
}

// readAnswer reads what encodeAnswer wrote: the task's value, or its error.
func readAnswer(data []byte) (object.Object, *object.Error, error) {
	if len(data) == 0 {
		return nil, nil, errCorrupt
	}
	value, _, err := decode(data[1:])
	if err != nil {
		return nil, nil, err
	}
	if data[0] == answerValue {
		return value, nil, nil
	}
	h, ok := value.(*object.Hash)
	if !ok {
		return nil, nil, errCorrupt
	}
	get := func(key string) object.Object {
		v, _ := h.Get(&object.String{Value: key})
		return v
	}
	e := &object.Error{Message: stringOf(get("message")), Source: stringOf(get("source"))}
	if line, ok := get("line").(*object.Integer); ok {
		e.Line = int(line.Value)
	}
	if column, ok := get("column").(*object.Integer); ok {
		e.Column = int(column.Value)
	}
	return nil, e, nil
}

// prefixWriter writes to out with prefix at the start of each line.
type prefixWriter struct {
	mu     sync.Mutex
	out    io.Writer
	prefix string
	start  bool // the next byte starts a line
}

func (w *prefixWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	var b bytes.Buffer
	for _, c := range p {
		if w.start {
			b.WriteString(w.prefix)
		}
		b.WriteByte(c)
		w.start = c == '\n'
	}
	if _, err := w.out.Write(b.Bytes()); err != nil {
		return 0, err
	}
	return len(p), nil
}

func isError(o object.Object) bool {
	_, ok := o.(*object.Error)
	return ok
}

// isFunction reports whether o can be called: a function of either engine
// or a builtin.
func isFunction(o object.Object) bool {
	switch o.(type) {
	case *object.Function, *object.Builtin, object.Callable:
		return true
	}
	return false
}

func stringOf(o object.Object) string {
	if s, ok := o.(*object.String); ok {
		return s.Value
	}
	return ""
}

func stringsOf(o object.Object) []string {
	list, ok := o.(*object.Array)
	if !ok {
		return nil
	}
	var out []string
	for _, item := range list.Elements {
		out = append(out, stringOf(item))
	}
	return out
}

func stringList(items []string) *object.Array {
	list := &object.Array{Elements: make([]object.Object, len(items))}
	for i, item := range items {
		list.Elements[i] = arg.Str(item)
	}
	return list
}
