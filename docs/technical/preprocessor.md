# Preprocessor (`internal/preproc`)

Sits between the lexer and the parser. Its only job is to expand file imports
and pass library imports through.

## Algorithm

1. Walk the token stream.
2. When `IMPORT STRING SEMI` is found:
   - Verify the string ends in `.j`.
   - Resolve the path (relative to the current file's directory, or absolute
     if it starts with `/`).
   - Reject if the path was already visited up the import chain (circular import).
   - Read the file, lex it (with file-tagged tokens), recursively preprocess it.
   - Splice the result (dropping the trailing `EOF`) at this point.
3. When `USE IDENT SEMI` is found: pass through unchanged. The parser turns
   it into an `ImportStmt` node.
4. Helpful errors for common mistakes:
   - `import IDENT;` (old library form) -> "use `use NAME;` for system libraries".
   - `import IDENT.j;` (old unquoted file form) -> "file imports take a string literal".
   - `use IDENT.j;` (file form with `use`) -> "use `import \"name.j\";` for files".

## Edge cases

- The path string must literally end in `.j`. `import "foo.go";` is rejected.
- Paths may contain `/` for subdirectories. Absolute paths are accepted as-is;
  relative paths resolve from the importing file's directory.
- Circular imports are detected by tracking absolute paths visited along the
  current chain.
