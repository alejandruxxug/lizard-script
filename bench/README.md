# Benchmarks: lizard-script next to Python

Each benchmark is written twice, as `name.liz` and `name.py`, doing the same work in the plainest way each language allows. Python uses `while` loops, the closest match to lizard's C-style `for`, so the comparison is about the engines, not about `range` running its counter in C.

```sh
python3 bench/compare.py           # builds lizardscript and prints a Markdown table
python3 bench/compare.py --runs 5
```

The script checks that lizard (on both engines) and Python print the same answer before it trusts a time. Times include starting the program, and each is the best of the runs.

To add a benchmark, add a pair of files whose first line is a comment saying what they measure, and that print one answer.

Startup is part of every time: about 26 ms for `lizardscript` (it installs the Terrarium) and 16 ms for Python on an M4, which matters for the short benchmarks.

## Why not compare with Go?

lizard-script runs on top of Go, so Go is the ceiling, not a peer: compiled Go does these loops roughly 50–100× faster than any bytecode interpreter. The fair comparison is with other interpreted languages, and CPython is the most familiar one.
