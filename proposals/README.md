# Lizard Enhancement Proposals (LEPs)

A **LEP** is how anyone proposes a change to the lizard-script language: new syntax, a change in what a construct does, a new builtin, or a new engine feature. It's a short Markdown file that explains the idea well enough to discuss it and, once accepted, to build it. (Bug fixes, docs and new Terrarium functions don't need one: see [CONTRIBUTING.md](../CONTRIBUTING.md).)

## How it works

1. **Idea.** Open an issue with the **Language proposal** template to see if others like it. Small ideas can stop here and go straight to a pull request if the maintainers say so.
2. **Draft.** Copy [`0000-template.md`](0000-template.md) to `NNNN-short-name.md` (the next free number) and open a pull request with it. Status: `Draft`.
3. **Discussion.** Comments on the pull request. Update the LEP as the design changes; it should always describe the current idea, with the alternatives that were rejected and why.
4. **Decision.** The maintainers (the lizard-script team) mark it `Accepted`, `Rejected` or `Deferred`, with the reason in the file, and merge it either way, so the reasoning stays on record.
5. **Implementation.** An accepted LEP is built in one or more pull requests (both engines, tests, docs, guide). When it lands, its status becomes `Final` and it gets a row in [DESIGN.md §7](../docs/DESIGN.md).

## What makes a good LEP

- A real problem, shown with a program that is awkward or impossible today.
- The smallest change that solves it, with examples of the new code.
- **What breaks:** which existing programs would behave differently (run them).
- How it fits the rest of the language (and both engines).

## Index

| LEP | Title | Status |
|---|---|---|
| [0001](0001-statement-ends-at-newline.md) | A statement ends at the end of its line | Final (accepted 2026-11-03, option 4) |
| — | *Ideas waiting for a LEP: named arguments (`f(angle: 90deg)`), classes for game objects, the parser's error recovery.* | |
