"""Runs each benchmark in lizard-script (VM and tree-walker), in Python and,
when `lua` is installed, in Lua; checks that they all print the same answer
(numbers compared to 1e-9), and prints a Markdown table.

    python3 bench/compare.py            # builds lizardscript from this repo
    python3 bench/compare.py --runs 5   # best of 5 instead of 3
    LIZARD=/path/to/lizardscript python3 bench/compare.py

Times include starting the program. Each number is the best of the runs.
"""

import argparse
import os
import platform
import re
import shutil
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


def numbers(text):
    """The numbers in an answer, so 2e+06 and 2000000.0 compare equal."""
    return [float(n) for n in re.findall(r"-?\d+\.?\d*(?:e[-+]?\d+)?", text)]


def same_answer(a, b):
    x, y = numbers(a), numbers(b)
    return len(x) == len(y) and all(abs(p - q) <= 1e-9 * max(1, abs(p)) for p, q in zip(x, y))


def lua_version(lua):
    done = subprocess.run([lua, "-v"], capture_output=True, text=True)
    return (done.stdout or done.stderr).split("  ")[0].strip()


def first_line(path):
    """The benchmark's description: its first comment line."""
    with open(path) as f:
        return f.readline().lstrip("-~>#").strip()


def verdict(other, vm, name):
    ratio = other / vm
    return f"lizard {ratio:.1f}× faster" if ratio >= 1 else f"{name} {1 / ratio:.1f}× faster"


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--runs", type=int, default=3)
    args = parser.parse_args()

    names = sorted(n[:-4] for n in os.listdir(HERE) if n.endswith(".liz"))
    with tempfile.TemporaryDirectory() as tmp:
        lizard = build_lizard(tmp)
        lua = shutil.which("lua")
        langs = f"Python {platform.python_version()}" + (f" · {lua_version(lua)}" if lua else "")
        print(f"{langs} · {platform.machine()} · best of {args.runs}\n")
        header = "| Benchmark | lizard VM | lizard tree | Python |" + (" Lua |" if lua else "") + " VM vs Python |" + (" VM vs Lua |" if lua else "")
        print(header)
        print("|" + "---|" * (header.count("|") - 1))
        for name in names:
            liz, py = os.path.join(HERE, name + ".liz"), os.path.join(HERE, name + ".py")
            vm, vm_out = best_time([lizard, "--vm", liz], args.runs)
            tree, tree_out = best_time([lizard, liz], args.runs)
            python, py_out = best_time([sys.executable, py], args.runs)
            answers = {"vm": vm_out, "tree": tree_out, "python": py_out}
            row = f"| **{name}**: {first_line(liz)} | {vm:.3f} s | {tree:.3f} s | {python:.3f} s |"
            lua_file = os.path.join(HERE, name + ".lua")
            if lua:
                if os.path.exists(lua_file):
                    lua_time, answers["lua"] = best_time([lua, lua_file], args.runs)
                    row += f" {lua_time:.3f} s |"
                else:
                    lua_time = None
                    row += " — |"
            if not all(same_answer(vm_out, a) for a in answers.values()):
                sys.exit(f"{name}: different answers: {answers}")
            row += f" {verdict(python, vm, 'Python')} |"
            if lua:
                row += f" {verdict(lua_time, vm, 'Lua') if lua_time else '—'} |"
            print(row)


if __name__ == "__main__":
    main()
