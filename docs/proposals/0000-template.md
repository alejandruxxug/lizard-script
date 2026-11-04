# LEP NNNN: Title

- **Author(s):** your name (@github)
- **Status:** Draft <!-- Draft · Accepted · Rejected · Deferred · Final -->
- **Created:** YYYY-MM-DD
- **Discussion:** link to the issue or pull request

## Summary

One paragraph: what changes, in plain words.

## Motivation

The problem. Show a program that is awkward, slow or impossible today.

```
-~> today
```

## The change

What the new code looks like, and exactly what it does. Include the grammar change (for docs/GRAMMAR.md) and the meaning (for docs/SEMANTICS.md), with examples and their output.

```
-~> with this LEP
```

## What breaks

Which existing programs would behave differently? Run `go test ./...` with a prototype, list any `.expected` file that would change, and say why it's worth it. "Nothing" is the best answer.

## Implementation

Which packages change (lexer, parser, ast, evaluator, compiler, vm, Terrarium), and how it works on both engines. Rough size.

## Alternatives

Other designs you considered, and why this one is better. Include "do nothing".

## Open questions

What still needs deciding.
