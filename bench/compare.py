"""Runs each benchmark in lizard-script (VM and tree-walker) and in Python,
checks that both print the same answer, and prints a Markdown table.

    python3 bench/compare.py            # builds lizardscript from this repo
    python3 bench/compare.py --runs 5   # best of 5 instead of 3
    LIZARD=/path/to/lizardscript python3 bench/compare.py

Times include starting the program. Each number is the best of the runs.
"""

import argparse
import os
import platform
import subprocess
import sys
import tempfile
import time

HERE = os.path.dirname(os.path.abspath(__file__))
REPO = os.path.dirname(HERE)


def build_lizard(out_dir):
    """Builds the lizardscript command from this repo, unless LIZARD is set."""
    if os.environ.get("LIZARD"):
        return os.environ["LIZARD"]
    exe = os.path.join(out_dir, "lizardscript")
    subprocess.run(["go", "build", "-o", exe, "."], cwd=REPO, check=True)
    return exe


def best_time(command, runs):
    """Runs command `runs` times; returns (best seconds, its output)."""
    best, output = None, None
    for _ in range(runs):
        start = time.perf_counter()
        done = subprocess.run(command, capture_output=True, text=True, check=True)
        took = time.perf_counter() - start
        if best is None or took < best:
            best, output = took, done.stdout.strip()
    return best, output


def first_line(path):
    """The benchmark's description: its first comment line."""
    with open(path) as f:
        return f.readline().lstrip("-~>#").strip()


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--runs", type=int, default=3)
    args = parser.parse_args()

    names = sorted(n[:-4] for n in os.listdir(HERE) if n.endswith(".liz"))
    with tempfile.TemporaryDirectory() as tmp:
        lizard = build_lizard(tmp)
        print(f"Python {platform.python_version()} · {platform.machine()} · best of {args.runs}\n")
        print("| Benchmark | lizard VM | lizard tree | Python | VM vs Python |")
        print("|---|---|---|---|---|")
        for name in names:
            liz, py = os.path.join(HERE, name + ".liz"), os.path.join(HERE, name + ".py")
            vm, vm_out = best_time([lizard, "--vm", liz], args.runs)
            tree, tree_out = best_time([lizard, liz], args.runs)
            python, py_out = best_time([sys.executable, py], args.runs)
            if not (vm_out == tree_out == py_out):
                sys.exit(f"{name}: different answers: vm {vm_out!r}, tree {tree_out!r}, python {py_out!r}")
            ratio = python / vm
            verdict = f"lizard {ratio:.1f}× faster" if ratio >= 1 else f"Python {1 / ratio:.1f}× faster"
            print(f"| **{name}**: {first_line(liz)} | {vm:.3f} s | {tree:.3f} s | {python:.3f} s | {verdict} |")


if __name__ == "__main__":
    main()
