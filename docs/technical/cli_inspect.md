# Inspection: `tokens` and `ast`

`cmd/jennifer/dump.go` and `cmd/jennifer/astjson.go` implement two
read-only inspection subcommands. `tokens` runs only the lexer and
prints one token per line in column-aligned `LINE:COL TYPE [lexeme]`
form - useful for tracing scanning issues and as a teaching tool.
`ast` runs lex + preproc + parse and writes the resulting AST as
two-space-indented JSON; every node carries `type`, `file`, `line`,
`col`, plus its node-specific fields.

The JSON emitter is hand-rolled in `astjson.go`'s `emitNode` (a switch
over every concrete AST type). We avoid `encoding/json` because its
reflect-based marshaling is fragile under TinyGo and at odds with the
tagged-union `Value` discipline used elsewhere; a switch over ~20 node
kinds is small enough to keep readable. Each field-emitter
(`emitStringField`, `emitBoolField`, `emitNodeListField`, etc.) writes
`"key": value,` and the closing `endObj` trims the trailing comma so
the output is valid JSON.


Part of the [CLI reference](cli.md).
