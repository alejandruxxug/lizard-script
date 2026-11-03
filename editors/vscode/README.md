# lizard-script for VS Code

Support for [lizard-script](https://github.com/alejandruxxug/lizard-script), the language so easy that even a lizard can understand it: `.liz` files (and `.lzdraw` drawings, which are lizard-script too).

- Highlighting, from the TextMate grammar, and from the language server where it knows more: functions, parameters, modules, builtins, colors and points.
- The mistakes `lizardscript check` finds, while you type, with the hint of each (`did you mean len?`).
- Format with `lizardscript fmt` (**Format Document**, or on save with `editor.formatOnSave`).
- Completion of the names in scope, the keywords, snippets, and members after a dot: `math.`, `text.`, `list.`, `hash.`, the Terrarium (`fs.`, `img.`, `window.`, ...), the standard modules (`lily2d.`, `lily3d.`) and hashes whose keys are written in the code. Each with its signature and its doc comment.
- Hover: signatures and doc comments, the value of a `let` with a literal, a literal's type (`90deg = 1.5708 rad`, a color's channels and a swatch, a point's coordinates).
- Signature help, go to definition, find references and rename (across the modules a file uses), the outline (functions, top-level lets and layers), workspace symbols, folding, and a color picker on every color literal.
- Commands: **lizard-script: Run file** (also the ▷ button of the editor), **Run file on the VM**, **Format**, **Check workspace** (to the Problems panel) and **Restart the language server**.

It needs the `lizardscript` command: build it from the repository (`go build -o lizardscript .`) and put it on your `PATH`, or set **lizard.path** to the binary. See [editors/README.md](https://github.com/alejandruxxug/lizard-script/blob/main/editors/README.md) for the rest.
