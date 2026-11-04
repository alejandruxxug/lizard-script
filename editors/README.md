# Editor support

Tools that help people write lizard-script in an editor. Nothing in this folder is part of the interpreter: `go build` and `go test ./...` don't look at it. The language server they start is: `lizardscript lsp` (package `tools/lsp/`).

| Folder | What it is |
| --- | --- |
| [`tree-sitter-lizard/`](#tree-sitter-tree-sitter-lizard) | the Tree-sitter grammar |
| [`vscode/`](#vs-code-vscode) | the VS Code extension, with the TextMate grammar in `vscode/syntaxes/` |
| [`zed/`](#zed-zed) | the Zed extension |

Neither extension is published to the VS Code Marketplace or the Zed registry: they are installed from this folder, as below.

## The language server: `lizardscript lsp`

`lizardscript lsp` speaks the [Language Server Protocol](https://microsoft.github.io/language-server-protocol/) on its standard input and output, so any editor with an LSP client can use it. It reads the code and the modules it uses; it never runs it.

| Feature | What it does |
| --- | --- |
| Diagnostics | the errors and warnings of `lizardscript check`, with the hint of each (`did you mean len?`), while you type (150 ms after the last key). Unused names and unreachable code are shown faded. |
| Formatting | `lizardscript fmt`, for the whole file. A file with a syntax error is left as it is. |
| Completion | the names the cursor can see (with lizard's late binding: a function sees a `let` further down, the same function doesn't), the names of the modules used without `as`, the builtins and the Terrarium, keywords and snippets (`function`, `if`, `for`, `while`, `layer`, `try`, `use`). After a dot: the members of `math`, `text`, `list`, `hash`, the Terrarium modules, a module used with `as` (also the standard ones: `lily2d.`, `lily3d.`) and a hash literal (`config.`). Each with its signature and its doc. |
| Hover | a function's signature and its `-~>` doc comment (read like `lizard.docs`: summary, `name: ...` parameter lines, `example:`); a builtin's documentation; a `let`'s value; a literal's type and worth: `90deg = 1.5708 rad`, `#3fa34d = rgb(63, 163, 77)` with a swatch, a point's coordinates. |
| Signature help | the parameters of the call the cursor is in, the current one marked. |
| Go to definition, references, highlights | for lets, functions, parameters, catch names, `use ... as` names, hash keys and the path of a `use` (it opens the module). A name of a module goes to the module's file; a standard module opens a copy of its text in the user's cache folder. References of a module's name include every file of the workspace that uses it. |
| Rename | scope-aware: an inner `let` or a parameter with the same name is a different name. A module's name is renamed in the module and in every file that uses it. Builtins, hash keys and standard modules can't be renamed. |
| Symbols | the outline of a file (functions, top-level lets, `use ... as` and layers, as a tree) and the workspace's symbols (every `.liz` file of its folders). |
| Folding | blocks, layers, hashes, arrays, calls over several lines, and runs of comment lines. |
| Semantic tokens | functions, parameters, modules, builtins (`defaultLibrary`), layer names (`label`), and two types of lizard's own: `color` (`#f2a65a`) and `point` (the brackets of `(x, y)`). |
| Colors | every color literal shows the editor's color picker; a color picked is written back as `#rrggbb` (`#rrggbbaa` with alpha) or as `rgb(...)`. |

The signatures and the text of the builtins come from `docs/SEMANTICS.md` and `docs/TERRARIUM.md`: `go generate ./tools/lsp` makes `tools/lsp/builtin_docs.go` from them, and a test fails when a document changed and the file was not made again.

The tests (`go test ./tools/lsp/...`) drive a server in the same process through the protocol, the way an editor does, on the files of `tools/lsp/testdata`. No editor is needed.

### Getting `lizardscript`

Both extensions start the `lizardscript` command. Build it from the repository and put it on your `PATH`:

```sh
go build -o lizardscript .
sudo mv lizardscript /usr/local/bin/        # or any folder on the PATH
lizardscript lsp < /dev/null                # ends at once: the server is there
```

or leave it anywhere and give its full path to the extension (`lizard.path` in VS Code, `lsp.lizardscript.binary.path` in Zed).

## VS Code (`vscode/`)

The extension (TypeScript) gives the language `lizard` for `.liz` files and `.lzdraw` drawings: the TextMate grammar, comments with `-~>`, brackets that close and color by pairs, indentation after `{`, `[`, `(` and `layer name`, snippets, an icon, and the language client that starts `lizardscript lsp`. Its commands (in the Command Palette, under **lizard-script**):

- **Run file**: saves the file and runs `lizardscript file.liz` in a terminal (also the ▷ button of the editor's title bar).
- **Run file on the VM**: `lizardscript --vm file.liz`.
- **Format**: formats the file (the same as **Format Document**; with `"editor.formatOnSave": true` it formats on save).
- **Check workspace**: runs `lizardscript check .` in each folder of the workspace, as a task whose findings go to the Problems panel.
- **Restart the language server**, after building a new `lizardscript`.

The setting **lizard.path** is the command (`lizardscript` by default). For files of the language the editor uses two spaces, as `fmt` does.

### Build and install the `.vsix`

You need [Node.js](https://nodejs.org) 18 or newer. In `editors/vscode`:

```sh
npm install                 # the client library and the build tools, in node_modules
npm run package             # type-checks, bundles src/ into out/extension.js and makes lizard-script.vsix
code --install-extension lizard-script.vsix
```

`npm run package` runs `vsce package` from `@vscode/vsce` (a dev dependency), which only makes the file: nothing is published. To work on the extension, open `editors/vscode` in VS Code and press F5: a second window starts with the extension loaded from the source (run `npm run build` after each change). `images/icon.liz` draws the icons with lily2d (`lizardscript icon.liz` in `images/`).

## Zed (`zed/`)

The extension has the language (`languages/lizard/config.toml`: `.liz` and `.lzdraw`, comments with `-~> `, brackets, two spaces), Zed's queries (`highlights`, `brackets`, `indents`, `outline`, `overrides`, written from the ones of `tree-sitter-lizard/queries` with the capture names Zed's themes know), and a Rust part (`src/lib.rs`, with `zed_extension_api`) that starts `lizardscript lsp`: the binary of the setting `lsp.lizardscript.binary.path`, or `lizardscript` on the `PATH`. `extension.toml` points Zed at the grammar: `editors/tree-sitter-lizard` of this repository on GitHub, at a commit that has it.

### Install it as a dev extension

You need Rust installed with [rustup](https://rustup.rs) (Zed builds the extension with it) and the `wasm32-wasip1` target:

```sh
rustup target add wasm32-wasip1
```

Then in Zed: open the command palette, run **zed: install dev extension**, and pick the folder `editors/zed`. Zed builds `src/lib.rs` to WebAssembly, fetches and builds the grammar, and loads the extension; it shows in the Extensions page as a dev extension. After changing the extension, **Rebuild** it there. Its log is in **zed: open log**.

To build it without Zed (to see that it compiles):

```sh
cd editors/zed
cargo build --release --target wasm32-wasip1
```

The queries must load with the grammar (a capture of a node or a field the grammar doesn't have is an error). With the Tree-sitter CLI, in `editors/tree-sitter-lizard`:

```sh
for q in ../zed/languages/lizard/*.scm; do tree-sitter query "$q" ../../examples/12_layers.liz > /dev/null || echo "$q"; done
```

To give Zed another `lizardscript`, in its `settings.json`:

```json
{
  "lsp": {
    "lizardscript": {
      "binary": { "path": "/full/path/to/lizardscript", "arguments": ["lsp"] }
    }
  }
}
```

When the grammar changes, `rev` in `extension.toml` must name a commit that has the change and is pushed: Zed fetches the grammar from GitHub. To try a grammar that is not pushed yet, set `repository` to `file:///full/path/to/lizard-script` and `rev` to a local commit.

## Other editors

Any editor with an LSP client can start `lizardscript lsp` for `.liz` files, and any editor built on Tree-sitter can use `tree-sitter-lizard`. In Neovim (0.11 or newer):

```lua
vim.filetype.add({ extension = { liz = "lizard", lzdraw = "lizard" } })
vim.lsp.config("lizardscript", { cmd = { "lizardscript", "lsp" }, filetypes = { "lizard" }, root_markers = { ".git" } })
vim.lsp.enable("lizardscript")
```

In Helix, in `languages.toml`:

```toml
[language-server.lizardscript]
command = "lizardscript"
args = ["lsp"]

[[language]]
name = "lizard"
scope = "source.lizard"
file-types = ["liz", "lzdraw"]
comment-token = "-~>"
indent = { tab-width = 2, unit = "  " }
language-servers = ["lizardscript"]
```

## Syntax grammars

Two grammars describe the syntax for editors. They only mark up the text (colors, folding, indentation); the parser in `lang/parser/` is still the one that decides what a program means and what is an error.

| Grammar | Folder | Used by |
| --- | --- | --- |
| Tree-sitter | `tree-sitter-lizard/` | Neovim, Helix, Zed, Emacs, and anything built on Tree-sitter |
| TextMate | `vscode/syntaxes/lizard.tmLanguage.json` | VS Code, and editors that read TextMate grammars (Sublime Text, GitHub's highlighter) |

The Tree-sitter grammar is the exact one: it builds a tree of the program. The TextMate grammar is a set of regular expressions per line, so it can't see across lines and is a little less precise. Where they differ, the Tree-sitter one is right.

### Tree-sitter (`tree-sitter-lizard/`)

It covers the whole language as in `docs/GRAMMAR.md`, [LEP 0001](../docs/proposals/0001-statement-ends-at-newline.md) and [LEP 0002](../docs/proposals/0002-modules.md): every statement and expression, numbers with exponents and units (`1e3`, `90deg`, `2px`), colors, points, strings with escapes, hashes, arrays, dot access, default parameters, `try` / `catch`, `use "path" as name` and `-~>` comments.

Three parts of the language need more than rules, and are done in the external scanner (`src/scanner.c`, plain C):

- **A statement ends at the end of its line.** The scanner gives the parser a `_terminator` at a line break, but only where the grammar can end a statement. After an operator, a comma or inside `(`, `[` and a hash `{`, no end is possible, so the line goes on by itself. `else` / `elif` at the start of the next line is not an end either. Before a `}` and at the end of the input the end is empty.
- **Layers by indentation.** `layer name` owns the following lines with their first token in a later column than the word `layer`. The scanner keeps the columns of the open layers and says when a body is over.
- **Words that are not reserved.** `layer`, `use`, `try` and `continue` are keywords only where the parser reads them as keywords (`layer name`, `use "x"`, `try {`, a lone `continue`), so `let layer = 1`, `use + 1` and `f(continue)` stay ordinary names, as in the language.

Rules of the language that are not in the grammar (and so are not errors here): `break` and `continue` must be inside a loop, a default parameter can't come before one without a default, a number can't be too large, and nesting can't be too deep. The same goes for a few messages the parser gives for things that are valid trees, like `5degx`. The parser keeps checking those.

Files:

- `grammar.js`: the grammar, with comments.
- `src/scanner.c`: the external scanner.
- `src/parser.c`, `src/grammar.json`, `src/node-types.json`: generated by `tree-sitter generate`. They are kept so that an editor can build the parser without the Tree-sitter CLI.
- `queries/`: `highlights.scm`, `indents.scm`, `folds.scm`, `locals.scm`, `brackets.scm` and `injections.scm` (empty on purpose: nothing is embedded in lizard-script).
- `test/corpus/*.txt`: the tests. `statements`, `expressions`, `literals`, `lines`, `layers`, `contextual_words` and `errors` are written by hand; `programs.txt` (every program of `tests/valid`, `examples` and `terrarium/lib`) and `error_programs.txt` (the syntax errors of `tests/errors`) are made by `test/generate-corpus.sh`.

To work on it, install the Tree-sitter CLI (a build tool, nothing is published) and a C compiler, then, in `editors/tree-sitter-lizard`:

```
npm install --global tree-sitter-cli     # or: npx tree-sitter-cli ...
tree-sitter generate                     # after changing grammar.js
tree-sitter test                         # runs test/corpus
tree-sitter parse ../../examples/12_layers.liz
sh test/generate-corpus.sh               # remakes the program tests; read the diff
```

Every `.liz` file of the repository parses without an `ERROR` node, except the programs that are made to be errors (`tests/errors`, `examples/errors`, and the fuzz case with a stray byte (`fuzz/testdata`)).

### TextMate (`vscode/syntaxes/lizard.tmLanguage.json`)

Scope name `source.lizard`, files `*.liz`. It colors keywords (`let`, `function`, `if` / `elif` / `else`, `while`, `for`, `break`, `return`, and `try` / `catch`, `continue`, `layer`, `use` / `as` where they are keywords), functions (declared names and calls), the builtins (`len`, `puts`, `push`, ...), the Terrarium modules (`math`, `text`, `list`, `hash`, `json`, `fs`, `os`, `time`, `img`, `clip`, `io`, `lizard`, `window`, `pen`), `true`, `false` and `null`, colors, points written with plain numbers, numbers with exponents and units, strings with escapes (a bad escape is marked), operators and `-~>` comments. A layer is only marked on its header line, because the body is found by indentation.

When a module or builtin is added to the Terrarium, add its name to `queries/highlights.scm` and to the `module` or `builtin` rule of the TextMate file.
