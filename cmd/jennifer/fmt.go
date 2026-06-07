// SPDX-License-Identifier: LGPL-3.0-only
// Copyright (C) 2026 <developer@mplx.eu>

package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mplx/jennifer-lang/internal/lexer"
)

// runFmt formats path's source to stdout per docs/stylespec.md. The
// formatter operates on the token stream rather than the AST so it can
// preserve `import "file.j";` statements verbatim (the preprocessor would
// otherwise inline them) and any parentheses the user wrote (the AST
// erases redundant grouping).
//
// Known v1 limitation: comments are dropped because the lexer strips them
// at scan time. This is documented in stylespec.md; preserving comments
// would require carrying them as tokens through the lexer.
func runFmt(path string) int {
	src, label, absPath, _, ok := loadProgramSource(path)
	if !ok {
		return 1
	}
	tokens, err := lexer.TokenizeWithFile(src, absPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", label, err.Error())
		printErrorContext(src, absPath, err)
		return 1
	}
	formatted := formatTokens(tokens)
	io.WriteString(os.Stdout, formatted)
	return 0
}

// formatTokens emits canonical Jennifer source from a complete token list
// (including the trailing EOF). Tests call this directly; the CLI wraps
// it with file I/O.
func formatTokens(tokens []lexer.Token) string {
	f := &fmtState{indent: 0}
	for i, t := range tokens {
		if t.Type == lexer.TOKEN_EOF {
			break
		}
		f.emit(t, peekAt(tokens, i+1))
	}
	return f.finish()
}

// fmtState tracks the running output and the bookkeeping needed to decide
// what separator (space / newline / nothing) to put between consecutive
// tokens. The driver loop calls `emit(curr, next)` for every non-EOF
// token, then `finish()` once for the trailing newline.
type fmtState struct {
	out         strings.Builder
	indent      int  // current block depth in indent units
	prev        lexer.Token
	hasPrev     bool // false before the first token
	atLineStart bool // true right after newline+indent has been written
	// Token kinds that "begin an operand context" - i.e. when the next
	// token is `-`, that `-` is binary, not unary. Maintained on every
	// emit() so unary-vs-binary disambiguation stays a state lookup.
	prevIsOperand     bool
	prevIsUnaryMinus  bool // true when prev was a `-` parsed as unary
}

const fmtIndent = "    " // 4 spaces, per stylespec.md

func (f *fmtState) emit(t, next lexer.Token) {
	// Closing braces dedent before they're written, so the `}` lands at
	// the outer indent level.
	if t.Type == lexer.TOKEN_RBRACE && f.indent > 0 {
		f.indent--
	}
	// Detect whether the `-` we're about to write is unary. A `-` is
	// unary when nothing operand-shaped came before it.
	isUnaryMinus := t.Type == lexer.TOKEN_MINUS && !f.prevIsOperand
	f.writeSeparator(t)
	f.writeToken(t, next)
	f.prev = t
	f.hasPrev = true
	f.atLineStart = false
	f.prevIsOperand = isOperandToken(t)
	f.prevIsUnaryMinus = isUnaryMinus
	if t.Type == lexer.TOKEN_LBRACE {
		f.indent++
	}
}

// writeSeparator decides what (if anything) goes between f.prev and t,
// and writes it. Five outcomes: nothing, single space, newline+indent,
// or - in special cases - the chosen separator overrides on either side.
func (f *fmtState) writeSeparator(t lexer.Token) {
	if !f.hasPrev {
		return
	}
	if f.atLineStart {
		return
	}
	// Statement terminator: ";" closes a statement; the next token starts
	// a new line at the current indent. Exception: the two `;`s inside
	// `for (...; ...; ...)` stay on the same line (handled by the
	// paren-depth check below).
	if f.prev.Type == lexer.TOKEN_SEMI {
		if !f.insideForHeader() {
			f.newline()
			return
		}
		f.out.WriteByte(' ')
		return
	}
	// Closing brace: blocks end with `}`; the next token starts a new
	// line, except for the cuddled `} else` / `} elseif` forms.
	if f.prev.Type == lexer.TOKEN_RBRACE {
		if t.Type == lexer.TOKEN_ELSE || t.Type == lexer.TOKEN_ELSEIF {
			f.out.WriteByte(' ')
			return
		}
		f.newline()
		return
	}
	// Opening brace: the body that follows always starts on a new line.
	// (Single-line block compaction is a future optional optimization;
	// for now the formatter is deterministic and always expands.)
	if f.prev.Type == lexer.TOKEN_LBRACE {
		f.newline()
		return
	}
	// No space between an IDENT or TYPE keyword and an opening `(` - call
	// sites (`printf(`) and type-conversion sites (`int(`) both use the
	// tight form. The leading keyword forms (`if (`, `while (`, `for (`,
	// `elseif (`) get a space, handled by the default case below since
	// those keyword types aren't in noSpaceBeforeLParen.
	if t.Type == lexer.TOKEN_LPAREN && noSpaceBeforeLParen(f.prev.Type) {
		return
	}
	// Tight punctuation: nothing between `(`/`[` and the next token, and
	// nothing between the previous token and `)`/`]`/`,`/`;`.
	switch t.Type {
	case lexer.TOKEN_RPAREN, lexer.TOKEN_COMMA, lexer.TOKEN_SEMI, lexer.TOKEN_DOT:
		return
	}
	if f.prev.Type == lexer.TOKEN_LPAREN || f.prev.Type == lexer.TOKEN_DOT {
		return
	}
	// Unary minus hugs its operand: `-5`, `-$x`, `-foo()`. The state
	// machine recorded on the previous emit() whether the `-` it just
	// wrote was unary; if so, no separator on its right side.
	if f.prevIsUnaryMinus {
		return
	}
	// Default: single space.
	f.out.WriteByte(' ')
}

// writeToken emits the token's text in its canonical form. Strings get
// their surrounding double quotes back; var refs get the `$` sigil.
func (f *fmtState) writeToken(t, _ lexer.Token) {
	switch t.Type {
	case lexer.TOKEN_VARREF:
		f.out.WriteByte('$')
		f.out.WriteString(t.Lexeme)
	case lexer.TOKEN_STRING:
		f.out.WriteString(quoteJenniferString(t.Lexeme))
	default:
		f.out.WriteString(canonicalLexeme(t))
	}
}

func (f *fmtState) newline() {
	f.out.WriteByte('\n')
	for i := 0; i < f.indent; i++ {
		f.out.WriteString(fmtIndent)
	}
	f.atLineStart = true
}

func (f *fmtState) finish() string {
	s := f.out.String()
	if !strings.HasSuffix(s, "\n") {
		s += "\n"
	}
	return s
}

// insideForHeader reports whether we're between the two `for (...;...;...)`
// semicolons. The check is approximate: it walks back through the output
// string looking for an unmatched `(` preceded by `for`. Cheap enough at
// formatter cadence and avoids carrying a token-aware paren stack.
//
// Used so the formatter writes a space (not a newline) after the two
// internal `for`-header semicolons.
func (f *fmtState) insideForHeader() bool {
	s := f.out.String()
	depth := 0
	// Walk backwards counting parens; a `(` with depth 0 is our enclosing
	// LPAREN. Check whether the keyword preceding it was `for`.
	for i := len(s) - 1; i >= 0; i-- {
		switch s[i] {
		case ')':
			depth++
		case '(':
			if depth == 0 {
				// Check the word that ends just before the `(`.
				j := i - 1
				for j >= 0 && s[j] == ' ' {
					j--
				}
				return j >= 2 && s[j-2:j+1] == "for"
			}
			depth--
		}
	}
	return false
}

// canonicalLexeme returns the source-form spelling of a token. For
// keywords and punctuation, the constant lexeme in the token is already
// canonical; for INT and FLOAT literals we use the lexeme as captured by
// the lexer (no normalization of float forms in v1).
func canonicalLexeme(t lexer.Token) string {
	if t.Lexeme != "" {
		return t.Lexeme
	}
	// Fallback for tokens whose lexeme field is empty (shouldn't normally
	// happen for anything we'd want to print, but keeps fmt total).
	return t.Type.String()
}

// quoteJenniferString re-quotes a string literal's *processed* value back
// into Jennifer-source form with double quotes and the standard escape
// sequences. Mirrors what the lexer's readString accepted on the way in.
func quoteJenniferString(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 2)
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString("\\\"")
		case '\\':
			b.WriteString("\\\\")
		case '\n':
			b.WriteString("\\n")
		case '\r':
			b.WriteString("\\r")
		case '\t':
			b.WriteString("\\t")
		case 0:
			b.WriteString("\\0")
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}

// isOperandToken reports whether t produces a value (so a following `-`
// is binary, not unary). The negation - "not an operand" - means t leaves
// the formatter in an expression-start context.
func isOperandToken(t lexer.Token) bool {
	switch t.Type {
	case lexer.TOKEN_INT, lexer.TOKEN_FLOAT, lexer.TOKEN_STRING,
		lexer.TOKEN_VARREF, lexer.TOKEN_TRUE, lexer.TOKEN_FALSE,
		lexer.TOKEN_NULL, lexer.TOKEN_IDENT,
		lexer.TOKEN_RPAREN, lexer.TOKEN_RBRACE:
		return true
	}
	return false
}

// noSpaceBeforeLParen lists the token types that hug a following `(`:
// function calls and type-conversion casts.
func noSpaceBeforeLParen(tt lexer.TokenType) bool {
	switch tt {
	case lexer.TOKEN_IDENT,
		lexer.TOKEN_INT_TYPE, lexer.TOKEN_FLOAT_TYPE,
		lexer.TOKEN_STRING_TYPE, lexer.TOKEN_BOOL_TYPE:
		return true
	}
	return false
}

func peekAt(tokens []lexer.Token, i int) lexer.Token {
	if i < 0 || i >= len(tokens) {
		return lexer.Token{}
	}
	return tokens[i]
}
