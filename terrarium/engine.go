package terrarium

// OnVM tells whether the program runs on the bytecode VM rather than the
// tree-walking evaluator. A host sets it before the run, like ProgramFile
// (repl.RunFile does); the task module runs its tasks on the same engine.
var OnVM bool
