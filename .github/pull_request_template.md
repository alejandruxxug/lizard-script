**What and why:**

**Checklist**
- [ ] `go test ./...` and `go vet ./...` pass; builds for macOS, Windows and Linux
- [ ] Both engines (evaluator and VM) give the same output, if the language changed
- [ ] Docs updated: DESIGN.md §7 row, GRAMMAR.md / SEMANTICS.md / TERRARIUM.md, and `docs/learn` (en + es) if users can see it
- [ ] Any behaviour change for existing programs is described above, with the `.expected` files updated
- [ ] Linked LEP, if this implements one
