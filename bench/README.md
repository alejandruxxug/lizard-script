# Benchmarks: lizard-script next to Python and Lua

Each benchmark is written as `name.liz`, `name.py` and `name.lua`, doing the same work in the plainest way each language allows. Python and Lua use `while` loops, the closest match to lizard's C-style `for`, so the comparison is about the engines, not about `range` or Lua's numeric `for` running the counter natively. The Lua column appears when `lua` is on the PATH.

```sh
python3 bench/compare.py           # builds lizardscript and prints a Markdown table
python3 bench/compare.py --runs 5
```

The script checks that lizard (on both engines), Python and Lua print the same answer (numbers compared to 1e-9) before it trusts a time. Times include starting the program, and each is the best of the runs.

To add a benchmark, add a pair of files whose first line is a comment saying what they measure, and that print one answer.

Startup is part of every time: about 9 ms for `lizardscript` and 16 ms for Python on an M4. It was 26 ms until the copy of Ebitengine in `third_party/ebiten` made its Metal device when the first window opens instead of when any program starts (see DESIGN.md section 8); the same command built without `terrarium/window` starts in 3.4 ms, the rest being the other `init`s of Ebitengine.

## Results

M4, Python 3.14.6, Lua 5.5.1, best of 5, 2026-11-03. Before is commit `5396133`; after is the VM with unboxed values, quickening and its fast loop (DESIGN.md section 2).

| Benchmark | VM before | VM after | tree after | Python | Lua | VM vs Python | VM vs Lua |
|---|---|---|---|---|---|---|---|
| closures | 0.109 s | 0.043 s | 0.417 s | 0.046 s | 0.014 s | lizard 1.1× faster | Lua 3.1× faster |
| fib | 0.073 s | 0.037 s | 0.270 s | 0.030 s | 0.008 s | Python 1.2× faster | Lua 4.6× faster |
| float | 0.304 s | 0.110 s | 1.602 s | 0.113 s | 0.024 s | lizard 1.0× faster | Lua 4.7× faster |
| hashes | 0.111 s | 0.094 s | 0.256 s | 0.051 s | 0.024 s | Python 1.9× faster | Lua 4.0× faster |
| loop | 0.613 s | 0.110 s | 3.027 s | 0.231 s | 0.054 s | lizard 2.1× faster | Lua 2.0× faster |
| points | 0.170 s | 0.118 s | 0.758 s | 0.079 s | 0.098 s | Python 1.5× faster | Lua 1.2× faster |
| strings | 0.711 s | 0.042 s | 0.157 s | 0.025 s | 0.211 s | Python 1.7× faster | lizard 5.0× faster |

The same, with the command built without the window package, so lizard starts in 3.4 ms instead of 28 (the tree-walker gained the faster strings and hash keys too; before, it was 0.817 s on strings):

| Benchmark | VM | tree | Python | Lua | VM vs Python | VM vs Lua |
|---|---|---|---|---|---|---|
| closures | 0.019 s | 0.381 s | 0.046 s | 0.014 s | lizard 2.4× faster | Lua 1.4× faster |
| fib | 0.013 s | 0.214 s | 0.028 s | 0.007 s | lizard 2.2× faster | Lua 1.7× faster |
| float | 0.084 s | 1.497 s | 0.112 s | 0.023 s | lizard 1.3× faster | Lua 3.6× faster |
| hashes | 0.068 s | 0.228 s | 0.049 s | 0.025 s | Python 1.4× faster | Lua 2.8× faster |
| loop | 0.080 s | 2.883 s | 0.230 s | 0.053 s | lizard 2.9× faster | Lua 1.5× faster |
| points | 0.077 s | 0.667 s | 0.077 s | 0.099 s | lizard 1.0× faster | lizard 1.3× faster |
| strings | 0.015 s | 0.117 s | 0.024 s | 0.205 s | lizard 1.6× faster | lizard 13.6× faster |

With the device made on the first window (M4, best of 5, 2026-11-03; before is commit `f34d1ac`, which started in 26 ms):

| Benchmark | VM before | VM after | tree after | Python | Lua | VM vs Python | VM vs Lua |
|---|---|---|---|---|---|---|---|
| closures | 0.041 s | 0.023 s | 0.394 s | 0.046 s | 0.014 s | lizard 2.0× faster | Lua 1.7× faster |
| fib | 0.036 s | 0.019 s | 0.237 s | 0.027 s | 0.007 s | lizard 1.5× faster | Lua 2.6× faster |
| float | 0.106 s | 0.089 s | 1.553 s | 0.110 s | 0.023 s | lizard 1.2× faster | Lua 3.8× faster |
| hashes | 0.089 s | 0.069 s | 0.232 s | 0.048 s | 0.023 s | Python 1.4× faster | Lua 3.0× faster |
| loop | 0.104 s | 0.093 s | 2.989 s | 0.222 s | 0.052 s | lizard 2.4× faster | Lua 1.8× faster |
| points | 0.108 s | 0.093 s | 0.719 s | 0.077 s | 0.097 s | Python 1.2× faster | lizard 1.1× faster |
| strings | 0.039 s | 0.019 s | 0.127 s | 0.023 s | 0.203 s | lizard 1.2× faster | lizard 10.6× faster |

What is left behind CPython is startup (above) and hashes, whose Go maps are large for a few keys; DESIGN.md section 8 has both.

The Go benchmarks of the VM (`go test ./vm -bench . -benchmem`, best of 3, each run compiling its program), before and after, with the tree-walker's for comparison:

| Benchmark | VM before | VM after | | allocs before → after | tree before | tree after |
|---|---|---|---|---|---|---|
| Fib25 | 13.89 ms | 3.45 ms | 4.0× | 125 → 188 | 85.59 ms | 81.54 ms |
| NestedLoop | 7.60 ms | 3.92 ms | 1.9× | 148703 → 86888 | 33.81 ms | 33.83 ms |
| StringBuild | 525 µs | 155 µs | 3.4× | 4108 → 2207 | 817 µs | 501 µs |
| ArrayPushIndex | 3.82 ms | 2.96 ms | 1.3× | 63681 → 31961 | 13.14 ms | 13.21 ms |
| HashGetSet | 958 µs | 670 µs | 1.4× | 5189 → 5282 | 3.15 ms | 3.17 ms |
| ClosureCalls | 3.11 ms | 1.82 ms | 1.7× | 43719 → 20230 | 12.41 ms | 12.35 ms |
| DotAccess | 1.35 ms | 338 µs | 4.0× | 30651 → 311 | 10.47 ms | 10.66 ms |
| FloatMath | 1.55 ms | 282 µs | 5.5× | 46979 → 278 | 10.78 ms | 10.73 ms |
| PointMath | 3.12 ms | 2.73 ms | 1.1× | 91998 → 80290 | 13.89 ms | 13.71 ms |
| BuiltinLoop | 4.14 ms | 3.00 ms | 1.4× | 127720 → 91112 | 19.02 ms | 18.43 ms |
| TopLevelDots | 2.45 ms | 1.94 ms | 1.3× | 30616 → 18884 | 10.43 ms | 10.37 ms |
| Build10kPoints | 2.59 ms | 2.28 ms | 1.1× | 62027 → 50338 | 10.54 ms | 10.77 ms |
| HashEquality | 1.42 ms | 1.21 ms | 1.2× | 272 → 432 | 3.09 ms | 3.11 ms |
| Hashes10k | 14.09 ms | 10.65 ms | 1.3× | 231786 → 108465 | 50.17 ms | 49.66 ms |
| Hashes10kLib | 7.11 ms | 6.12 ms | 1.2× | 121672 → 78274 | 22.67 ms | 22.21 ms |
| Geometry | 6.57 ms | 6.37 ms | 1.0× | 142272 → 110953 | 41.45 ms | 41.80 ms |

`TokenScan` came later, with the index caches, the string comparisons and `push` of DESIGN.md section 2 (*Index caches*): an editor's scanner over 22,000 tokens, 9.60 ms before them (commit `f34d1ac`) and 6.37 ms after, 13822 allocations (tree: 73.2 ms). The others stayed within 2%, or got faster where they push or index (`ArrayPushIndex` 2.89 → 2.57 ms, `BuiltinLoop` 2.96 → 2.64 ms, `Hashes10kLib` 6.16 → 5.78 ms).


## Why not compare with Go?

lizard-script runs on top of Go, so Go is the ceiling, not a peer: compiled Go does these loops roughly 50–100× faster than any bytecode interpreter. The fair comparison is with other interpreted languages: CPython is the most familiar one, and Lua 5.x (a small register-based VM in C with unboxed numbers) is the classic fast one.
