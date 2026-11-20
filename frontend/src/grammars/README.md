# Grammars

Tree-sitter grammars (wasm) and highlight queries for syntax highlighting in
diffs (`lib/highlight.ts`). Colours are in `themes/syntax.css`.

| Language | Grammar (wasm) | Highlight query |
| --- | --- | --- |
| Go, Python, JavaScript, TypeScript, TSX, Bash, C, C++, JSON | `tree-sitter-*` on npm (tree-sitter org), MIT | the grammar's `queries/highlights.scm`; TypeScript adds JavaScript's, TSX also JSX's, C++ starts with C's |
| YAML, TOML | `@tree-sitter-grammars/tree-sitter-*` on npm, MIT | YAML: the grammar's; TOML: zed-extensions/toml, Apache-2.0 |
| Make | `tree-sitter-make` on npm, MIT | caius/zed-make, MIT |
| Markdown, Markdown inline | tree-sitter-grammars/tree-sitter-markdown v0.5.4 release, MIT | the grammar's |
| HTML | `tree-sitter-html` on npm, MIT | zed-industries/zed `extensions/html`, Apache-2.0 |
| SQL | DerekStride/tree-sitter-sql 0.3.13, MIT, built with `tree-sitter build --wasm` | zed-extensions/sql, MIT |

Zed's own queries for the other languages are GPL-3.0, so they are not used.
