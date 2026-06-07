# Jennifer Interpreter - Technical Documentation

Internals of the Jennifer interpreter. This directory is split by topic;
each page is small enough to read in one sitting.

## Contents

- [Lexer](lexer.md) - hand-written scanner, token kinds, position tracking,
  identifier rules.
- [Grammar and parser](grammar.md) - the EBNF the parser accepts, plus the
  AST node table and parser implementation notes.
- [Preprocessor](preprocessor.md) - how `import "file.j";` is expanded
  before the parser runs.
- [Interpreter](interpreter.md) - runtime values, scoped environment,
  execution model, library/builtins, error model.
- [CLI](cli.md) - subcommands (`run`, `repl`, `tokens`, `ast`, `fmt`,
  `version`, `help`), the REPL, the inspection dumps, the formatter,
  and version injection.
- [Testing](testing.md) - which packages test what.
- [File map](filemap.md) - one-line description of every file in the
  repository.
- [Rejected features](rejected.md) - proposals that were turned down,
  with the reasoning so they don't come back.
- [TinyGo notes](tinygo.md) - the constraints TinyGo imposes and how the
  codebase respects them.

## Pipeline

The compiler/interpreter is a five-stage pipeline:

```
   source (string)
       │
       ▼
   ┌────────┐
   │ lexer  │   internal/lexer
   └────────┘
       │ []Token
       ▼
   ┌──────────────┐
   │ preprocessor │   internal/preproc   (splices file imports)
   └──────────────┘
       │ []Token
       ▼
   ┌────────┐
   │ parser │   internal/parser
   └────────┘
       │ *Program (AST)
       ▼
   ┌─────────────┐
   │ interpreter │   internal/interpreter + internal/lib/io (and other libs)
   └─────────────┘
       │
       ▼
     stdout / runtime error
```

The CLI lives in `cmd/jennifer/main.go` and orchestrates these stages.
For the user-facing language reference (types, control flow, libraries),
see [../user-guide.md](../user-guide.md). For the style rules `jennifer
fmt` enforces, see [../stylespec.md](../stylespec.md).
