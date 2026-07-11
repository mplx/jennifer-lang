# Inspection: `tokens` and `ast`

`cmd/jennifer/dump.go` and `cmd/jennifer/astjson.go` implement two
read-only inspection subcommands over the front of the pipeline: `tokens`
stops after the lexer, `ast` after the parser. They make the two
intermediate representations the interpreter builds visible, which is as
much a teaching aid as a debugging one. Both live only in the default
`jennifer` binary (the run-only `jennifer-tiny` build stubs them).

The examples below use this three-line program, `snippet.j`:

```jennifer
use io;
def x as int init 41;
io.printf("%d\n", $x + 1);
```

## `tokens` - the lexer's stream

`tokens` runs only the lexer and prints one token per line in
column-aligned `LINE:COL TYPE [lexeme]` form - useful for tracing a
scanning issue:

```text
$ jennifer tokens snippet.j
1:1   USE
1:5   IDENT     "io"
1:7   SEMI
2:1   DEF
2:5   IDENT     "x"
2:7   AS
2:10  INT_TYPE
2:14  INIT
2:19  INT       "41"
2:21  SEMI
3:1   IDENT     "io"
3:3   DOT
3:4   IDENT     "printf"
3:10  LPAREN
3:11  STRING    "%d\n"
3:17  COMMA
3:19  VARREF    "x"
3:22  PLUS
3:24  INT       "1"
3:25  RPAREN
3:26  SEMI
4:1   EOF
```

A few things the stream makes concrete: every token records its source
`LINE:COL`; the type keyword `int` scans to its own `INT_TYPE`, distinct
from the `INT` literal `41`; the `$x` use-site is a single `VARREF "x"`
with the sigil already consumed; and the stream always terminates in
`EOF`.

## `ast` - the parsed tree as JSON

`ast` runs lex + preproc + parse and writes the AST as two-space-indented
JSON. Every node carries `type`, `file`, `line`, `col`, plus its
node-specific fields:

```text
$ jennifer ast snippet.j
```

```json
{
  "type": "Program",
  "line": 1,
  "col": 1,
  "imports": [
    { "type": "ImportStmt", "line": 1, "col": 1, "name": "io" }
  ],
  "moduleImports": [],
  "methods": [],
  "topLevel": [
    {
      "type": "DefineStmt",
      "line": 2, "col": 1,
      "isConst": false,
      "exported": false,
      "varName": "x",
      "varType": "int",
      "init": { "type": "IntLit", "line": 2, "col": 19, "value": 41 }
    },
    {
      "type": "ExprStmt",
      "line": 3, "col": 1,
      "expr": {
        "type": "QualifiedCallExpr",
        "line": 3, "col": 1,
        "prefix": "io",
        "callee": "printf",
        "args": [
          { "type": "StringLit", "line": 3, "col": 11, "value": "%d\n" },
          {
            "type": "BinaryExpr", "line": 3, "col": 19, "op": "+",
            "left":  { "type": "VarExpr", "line": 3, "col": 19, "name": "x" },
            "right": { "type": "IntLit",  "line": 3, "col": 24, "value": 1 }
          }
        ]
      }
    }
  ]
}
```

Each node also carries a `file` field (the resolved absolute source path),
elided above for width. Because preproc runs before the parse, any
`include`d file is already spliced and `import` statements are resolved, so
the tree is exactly what the interpreter walks: `$x + 1` as a `BinaryExpr`
over a `VarExpr` and an `IntLit`, and `io.printf(...)` as a single
`QualifiedCallExpr` with a `prefix` / `callee` pair.

## Implementation

The JSON emitter is hand-rolled in `astjson.go`'s `emitNode` (a switch
over every concrete AST type). We avoid `encoding/json` because its
reflect-based marshaling is fragile under TinyGo and at odds with the
tagged-union `Value` discipline used elsewhere; a switch over ~20 node
kinds is small enough to keep readable. Each field-emitter
(`emitStringField`, `emitBoolField`, `emitNodeListField`, etc.) writes
`"key": value,` and the closing `endObj` trims the trailing comma so
the output is valid JSON.


Part of the [CLI reference](cli.md).
