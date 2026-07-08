# Formatter (`cmd/jennifer/fmt.go`)

`jennifer fmt` formats source per [../user-guide/style-guide.md](../user-guide/style-guide.md).
It operates on the lexer's token stream rather than the AST, for two
reasons:

1. **`import "file.j";` survives.** The preprocessor consumes file
   imports before the parser sees them; an AST-based formatter would
   inline every import, which is the opposite of what a developer
   wants from `fmt`. The token-level formatter sees IMPORT tokens
   unchanged and re-emits them.
2. **User-written parens survive.** The AST records grouping only
   through nesting structure; redundant parens are erased. A token
   walker preserves LPAREN/RPAREN exactly as written, so
   `($a + $b) * $c` stays parenthesized after a round trip.

`formatTokens(tokens)` drives a small state machine (`fmtState`): for
each token it computes the separator (`writeSeparator`) - none, a
space, or a newline-plus-indent - and then writes the token's canonical
spelling (`writeToken`). Key state fields:

- `indent` bumps on `{` and drops on `}` (the closing brace dedents
  *before* it's written so it lands at the outer indent).
- `prevIsOperand` answers "is the next `-` binary or unary?" - flipped
  by `isOperandToken` after every emit.
- `prevIsUnaryMinus` suppresses the right-side space after a `-` that
  was determined to be unary.
- `insideForHeader` is a small backward scan that lets the two `;`s
  inside `for (...; ...; ...)` stay on the same line.

Strings are re-quoted with `quoteJenniferString` (double quotes plus
standard escapes), mirroring the lexer's `readString` on the way in.

Comments and blank lines survive a `fmt` round-trip: the lexer
emits them as trivia tokens, and `emitTrivia` writes them inline
without disturbing the surrounding state machine. Leading
comments land on their own line at the current indent; trailing
same-line comments stay on the same line; runs of blank lines
collapse to one. Block comments may nest.


Part of the [CLI reference](cli.md).
