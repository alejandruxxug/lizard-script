package vm

import (
	"fmt"
	"io"
	"lizardscript/code"
	"lizardscript/object"
	"sort"
	"time"
)

// The profiler counts every call the VM makes while it is on: how many
// times each function and builtin ran, the time spent inside it (total)
// and the part of that not spent in the calls it made (self). It measures
// each call with the clock rather than sampling, so short programs get
// exact counts; while it is off, a call costs one more nil check.

// Profile is what the profiler measured between StartProfile and
// StopProfile.
type Profile struct {
	// Functions are the lizard functions (and the program itself), and
	// Builtins the calls to anything else: builtins and the libraries'
	// functions. Both are sorted by self time, the most first.
	Functions []ProfileRow
	Builtins  []ProfileRow
	// Duration is the time the profiler was on.
	Duration time.Duration
	// File is shown for the functions whose source has no name (a program
	// read with lexer.New): the host sets it to the file it ran.
	File string
}

// ProfileRow is one function or builtin of a Profile.
type ProfileRow struct {
	// Name is the function's name, "function@line" for an anonymous one,
	// "program" for the program's own code, or the builtin's name.
	Name string
	// Source and Line are where a lizard function starts; "" and 0 for a
	// builtin.
	Source string
	Line   int
	Calls  int
	// Total is the time inside the function, counted once for recursive
	// calls; Self leaves out the time of the calls it made.
	Self, Total time.Duration
}

// profiler is the profile being taken, nil while profiling is off. Like
// the rest of the VM it is for one goroutine.
var profiler *profileState

type profileState struct {
	started time.Time
	rows    map[any]*profileRow
	// open are the calls running now, innermost last, across every VM run
	// (a builtin like list.sort starts one per call).
	open []openCall
}

type profileRow struct {
	ProfileRow
	builtin bool
	active  int // the open calls of this row, for recursive functions
}

type openCall struct {
	row      *profileRow
	start    time.Time
	children time.Duration
}

// StartProfile starts profiling every VM run until StopProfile. A profile
// already running starts over.
func StartProfile() {
	profiler = &profileState{started: time.Now(), rows: map[any]*profileRow{}}
}

// StopProfile stops profiling and returns what was measured. Calls still
// running are left out. Without StartProfile it returns an empty profile.
func StopProfile() Profile {
	p := profiler
	profiler = nil
	if p == nil {
		return Profile{}
	}
	profile := Profile{Duration: time.Since(p.started)}
	for _, row := range p.rows {
		if row.builtin {
			profile.Builtins = append(profile.Builtins, row.ProfileRow)
		} else {
			profile.Functions = append(profile.Functions, row.ProfileRow)
		}
	}
	sortRows(profile.Functions, bySelf)
	sortRows(profile.Builtins, bySelf)
	return profile
}

// programKey is the row of the programs' own code (outside functions).
type programKey struct{}

// enterFunction counts a call of a compiled function starting now. Rows
// are by source, so a program compiled again keeps its rows.
func (p *profileState) enterFunction(fn *code.Function) {
	if fn.Literal == nil {
		p.enter(p.row(programKey{}, ProfileRow{Name: "program", Line: 1}, false))
		return
	}
	lit := fn.Literal
	name := fn.Name
	if name == "" {
		name = fmt.Sprintf("function@%d", lit.Token.Line)
	}
	p.enter(p.row(lit, ProfileRow{Name: name, Source: lit.Token.Source, Line: lit.Token.Line}, false))
}

// enterForeign counts a call of anything that is not a VM closure: a
// builtin, or a function of the evaluator (a lizard function too, shown
// by the line of its body).
func (p *profileState) enterForeign(callee object.Object) {
	switch callee := callee.(type) {
	case *object.Builtin:
		p.enter(p.row(callee.Name, ProfileRow{Name: callee.Name}, true))
	case *object.Function:
		tok := callee.Body.Token
		row := ProfileRow{Name: fmt.Sprintf("function@%d", tok.Line), Source: tok.Source, Line: tok.Line}
		p.enter(p.row(callee.Body, row, false))
	default:
		name := string(callee.Type())
		p.enter(p.row(name, ProfileRow{Name: name}, true))
	}
}

// row is the row of key, made from first the first time.
func (p *profileState) row(key any, first ProfileRow, builtin bool) *profileRow {
	row := p.rows[key]
	if row == nil {
		row = &profileRow{ProfileRow: first, builtin: builtin}
		p.rows[key] = row
	}
	return row
}

func (p *profileState) enter(row *profileRow) {
	row.Calls++
	row.active++
	p.open = append(p.open, openCall{row: row, start: time.Now()})
}

// leave ends the innermost open call.
func (p *profileState) leave() {
	n := len(p.open)
	if n == 0 {
		return
	}
	call := p.open[n-1]
	p.open = p.open[:n-1]
	elapsed := time.Since(call.start)
	call.row.Self += elapsed - call.children
	call.row.active--
	if call.row.active == 0 {
		call.row.Total += elapsed
	}
	if n > 1 {
		p.open[n-2].children += elapsed
	}
}

// leaveFrames ends the profiled calls of the frames from depth up, when a
// return, an error or a try leaves them.
func (vm *VM) leaveFrames(depth int) {
	for i := len(vm.frames) - 1; i >= depth; i-- {
		if p := vm.frames[i].profile; p != nil && p == profiler {
			p.leave()
		}
	}
}

// Writing a profile

const profileRows = 25

func bySelf(a, b ProfileRow) bool  { return a.Self > b.Self }
func byTotal(a, b ProfileRow) bool { return a.Total > b.Total }

func sortRows(rows []ProfileRow, less func(a, b ProfileRow) bool) {
	sort.SliceStable(rows, func(i, j int) bool {
		if less(rows[i], rows[j]) != less(rows[j], rows[i]) {
			return less(rows[i], rows[j])
		}
		return rows[i].Name < rows[j].Name
	})
}

// Write prints the profile as three tables: the top functions by self
// time, the same by total time, and the top builtins.
func (p Profile) Write(w io.Writer) {
	fmt.Fprintf(w, "profile: %s\n", p.Duration.Round(time.Microsecond))
	p.writeTable(w, "functions by self time", p.Functions, bySelf)
	p.writeTable(w, "functions by total time", p.Functions, byTotal)
	p.writeTable(w, "builtins by self time", p.Builtins, bySelf)
}

func (p Profile) writeTable(w io.Writer, title string, rows []ProfileRow, less func(a, b ProfileRow) bool) {
	sorted := append([]ProfileRow{}, rows...)
	sortRows(sorted, less)
	if len(sorted) > profileRows {
		sorted = sorted[:profileRows]
	}
	fmt.Fprintf(w, "\n%s\n", title)
	fmt.Fprintf(w, "%12s %12s %10s  %s\n", "self", "total", "calls", "name")
	for _, row := range sorted {
		fmt.Fprintf(w, "%12s %12s %10d  %s\n", duration(row.Self), duration(row.Total), row.Calls, p.label(row))
	}
}

// label is name (file:line) for a lizard function, the name for a builtin.
func (p Profile) label(row ProfileRow) string {
	if row.Line == 0 {
		return row.Name
	}
	source := row.Source
	if source == "" {
		source = p.File
	}
	if source == "" {
		return fmt.Sprintf("%s (line %d)", row.Name, row.Line)
	}
	return fmt.Sprintf("%s (%s:%d)", row.Name, source, row.Line)
}

func duration(d time.Duration) string {
	return fmt.Sprintf("%.3fms", float64(d)/float64(time.Millisecond))
}
