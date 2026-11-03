# Benchmarks: lizard-script next to Python and Lua

Each benchmark is written as `name.liz`, `name.py` and `name.lua`, doing the same work in the plainest way each language allows. Python and Lua use `while` loops, the closest match to lizard's C-style `for`, so the comparison is about the engines, not about `range` or Lua's numeric `for` running the counter natively. The Lua column appears when `lua` is on the PATH.

```sh
python3 bench/compare.py           # builds lizardscript and prints a Markdown table
python3 bench/compare.py --runs 5
```

The script checks that lizard (on both engines), Python and Lua print the same answer (numbers compared to 1e-9) before it trusts a time. Times include starting the program, and each is the best of the runs.

To add a benchmark, add a pair of files whose first line is a comment saying what they measure, and that print one answer.

Startup is part of every time: about 26 ms for `lizardscript` (it installs the Terrarium) and 16 ms for Python on an M4, which matters for the short benchmarks.

## Why not compare with Go?

lizard-script runs on top of Go, so Go is the ceiling, not a peer: compiled Go does these loops roughly 50–100× faster than any bytecode interpreter. The fair comparison is with other interpreted languages: CPython is the most familiar one, and Lua 5.x (a small register-based VM in C with unboxed numbers) is the classic fast one.
