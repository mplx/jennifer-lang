// SPDX-License-Identifier: LGPL-3.0-only
// Copyright (C) 2026 <developer@mplx.eu>

package lexer

import (
	"fmt"
	"strings"
	"unicode"
)

// Lexer turns a Jennifer source string into a stream of tokens.
// It tracks line and column for error reporting; column is 1-based.
// File is attached to every produced token for cross-file diagnostics.
type Lexer struct {
	src  []rune
	pos  int
	line int
	col  int
	file string
}

func New(source string) *Lexer {
	return &Lexer{src: []rune(source), pos: 0, line: 1, col: 1}
}

// NewWithFile is like New but tags every produced token with the given file name.
func NewWithFile(source, file string) *Lexer {
	l := New(source)
	l.file = file
	return l
}

// LexError carries a source position so the parser/interpreter can surface useful messages.
type LexError struct {
	Msg  string
	File string
	Line int
	Col  int
}

func (e *LexError) Error() string {
	if e.File != "" {
		return fmt.Sprintf("lex error at %s:%d:%d: %s", e.File, e.Line, e.Col, e.Msg)
	}
	return fmt.Sprintf("lex error at %d:%d: %s", e.Line, e.Col, e.Msg)
}

// Position implements the positioned-error interface used by the CLI to
// extract a file/line/col without parsing the error string.
func (e *LexError) Position() (file string, line, col int) {
	return e.File, e.Line, e.Col
}

// Tokenize runs the lexer to completion and returns the full token list (terminated by TOKEN_EOF).
func Tokenize(source string) ([]Token, error) {
	return TokenizeWithFile(source, "")
}

// TokenizeWithFile is like Tokenize but tags every produced token with the file name.
// Use it when the source comes from a known file so cross-file diagnostics work.
func TokenizeWithFile(source, file string) ([]Token, error) {
	l := NewWithFile(source, file)
	var out []Token
	for {
		tok, err := l.Next()
		if err != nil {
			return nil, err
		}
		if file != "" {
			tok.File = file
		}
		out = append(out, tok)
		if tok.Type == TOKEN_EOF {
			return out, nil
		}
	}
}

// Next returns the next token. At end of input it repeatedly returns TOKEN_EOF.
func (l *Lexer) Next() (Token, error) {
	if err := l.skipWhitespaceAndComments(); err != nil {
		return Token{}, err
	}
	if l.pos >= len(l.src) {
		return Token{Type: TOKEN_EOF, Line: l.line, Col: l.col}, nil
	}

	startLine, startCol := l.line, l.col
	ch := l.src[l.pos]

	switch {
	case ch == '{':
		l.advance()
		return Token{Type: TOKEN_LBRACE, Lexeme: "{", Line: startLine, Col: startCol}, nil
	case ch == '}':
		l.advance()
		return Token{Type: TOKEN_RBRACE, Lexeme: "}", Line: startLine, Col: startCol}, nil
	case ch == '(':
		l.advance()
		return Token{Type: TOKEN_LPAREN, Lexeme: "(", Line: startLine, Col: startCol}, nil
	case ch == ')':
		l.advance()
		return Token{Type: TOKEN_RPAREN, Lexeme: ")", Line: startLine, Col: startCol}, nil
	case ch == '[':
		l.advance()
		return Token{Type: TOKEN_LBRACKET, Lexeme: "[", Line: startLine, Col: startCol}, nil
	case ch == ']':
		l.advance()
		return Token{Type: TOKEN_RBRACKET, Lexeme: "]", Line: startLine, Col: startCol}, nil
	case ch == ';':
		l.advance()
		return Token{Type: TOKEN_SEMI, Lexeme: ";", Line: startLine, Col: startCol}, nil
	case ch == ':':
		l.advance()
		return Token{Type: TOKEN_COLON, Lexeme: ":", Line: startLine, Col: startCol}, nil
	case ch == ',':
		l.advance()
		return Token{Type: TOKEN_COMMA, Lexeme: ",", Line: startLine, Col: startCol}, nil
	case ch == '=':
		l.advance()
		if next, ok := l.peek(0); ok && next == '=' {
			l.advance()
			return Token{Type: TOKEN_EQ, Lexeme: "==", Line: startLine, Col: startCol}, nil
		}
		return Token{Type: TOKEN_ASSIGN, Lexeme: "=", Line: startLine, Col: startCol}, nil
	case ch == '<':
		l.advance()
		if next, ok := l.peek(0); ok && next == '=' {
			l.advance()
			return Token{Type: TOKEN_LE, Lexeme: "<=", Line: startLine, Col: startCol}, nil
		}
		if next, ok := l.peek(0); ok && next == '<' {
			l.advance()
			return Token{Type: TOKEN_SHL, Lexeme: "<<", Line: startLine, Col: startCol}, nil
		}
		return Token{Type: TOKEN_LT, Lexeme: "<", Line: startLine, Col: startCol}, nil
	case ch == '>':
		l.advance()
		if next, ok := l.peek(0); ok && next == '=' {
			l.advance()
			return Token{Type: TOKEN_GE, Lexeme: ">=", Line: startLine, Col: startCol}, nil
		}
		if next, ok := l.peek(0); ok && next == '>' {
			l.advance()
			return Token{Type: TOKEN_SHR, Lexeme: ">>", Line: startLine, Col: startCol}, nil
		}
		return Token{Type: TOKEN_GT, Lexeme: ">", Line: startLine, Col: startCol}, nil
	case ch == '&':
		l.advance()
		return Token{Type: TOKEN_BIT_AND, Lexeme: "&", Line: startLine, Col: startCol}, nil
	case ch == '|':
		l.advance()
		return Token{Type: TOKEN_BIT_OR, Lexeme: "|", Line: startLine, Col: startCol}, nil
	case ch == '^':
		l.advance()
		return Token{Type: TOKEN_BIT_XOR, Lexeme: "^", Line: startLine, Col: startCol}, nil
	case ch == '~':
		l.advance()
		return Token{Type: TOKEN_BIT_NOT, Lexeme: "~", Line: startLine, Col: startCol}, nil
	case ch == '+':
		l.advance()
		return Token{Type: TOKEN_PLUS, Lexeme: "+", Line: startLine, Col: startCol}, nil
	case ch == '-':
		l.advance()
		return Token{Type: TOKEN_MINUS, Lexeme: "-", Line: startLine, Col: startCol}, nil
	case ch == '*':
		l.advance()
		return Token{Type: TOKEN_STAR, Lexeme: "*", Line: startLine, Col: startCol}, nil
	case ch == '/':
		l.advance()
		if next, ok := l.peek(0); ok && next == '/' {
			l.advance()
			return Token{Type: TOKEN_DIV, Lexeme: "//", Line: startLine, Col: startCol}, nil
		}
		return Token{Type: TOKEN_SLASH, Lexeme: "/", Line: startLine, Col: startCol}, nil
	case ch == '%':
		l.advance()
		return Token{Type: TOKEN_PERCENT, Lexeme: "%", Line: startLine, Col: startCol}, nil
	case ch == '.':
		l.advance()
		return Token{Type: TOKEN_DOT, Lexeme: ".", Line: startLine, Col: startCol}, nil
	case ch == '"' || ch == '\'':
		return l.readString(ch, startLine, startCol)
	case ch == '$':
		return l.readVarRef(startLine, startCol)
	case unicode.IsDigit(ch):
		return l.readNumber(startLine, startCol)
	case isIdentStart(ch):
		return l.readIdentifierOrKeyword(startLine, startCol)
	}

	return Token{}, &LexError{File: l.file, Msg: fmt.Sprintf("unexpected character %q", ch), Line: startLine, Col: startCol}
}

func (l *Lexer) advance() {
	if l.pos >= len(l.src) {
		return
	}
	if l.src[l.pos] == '\n' {
		l.line++
		l.col = 1
	} else {
		l.col++
	}
	l.pos++
}

func (l *Lexer) peek(offset int) (rune, bool) {
	idx := l.pos + offset
	if idx < 0 || idx >= len(l.src) {
		return 0, false
	}
	return l.src[idx], true
}

func (l *Lexer) skipWhitespaceAndComments() error {
	for l.pos < len(l.src) {
		ch := l.src[l.pos]
		if unicode.IsSpace(ch) {
			l.advance()
			continue
		}
		if ch == '#' {
			// `#` line comment - runs to end of line. Lets `#!/usr/bin/env -S
			// jennifer run` work as a shebang since the kernel reads the line
			// and the interpreter then sees it as a comment.
			for l.pos < len(l.src) && l.src[l.pos] != '\n' {
				l.advance()
			}
			continue
		}
		if ch == '/' {
			if next, ok := l.peek(1); ok && next == '*' {
				startLine, startCol := l.line, l.col
				l.advance() // /
				l.advance() // *
				for {
					if l.pos >= len(l.src) {
						return &LexError{File: l.file, Msg: "unterminated block comment", Line: startLine, Col: startCol}
					}
					if l.src[l.pos] == '*' {
						if next, ok := l.peek(1); ok && next == '/' {
							l.advance()
							l.advance()
							break
						}
					}
					l.advance()
				}
				continue
			}
			// Note: `//` is no longer a comment - it's the integer-division
			// operator (see the SLASH/SLASH dispatch in Next()).
		}
		return nil
	}
	return nil
}

func (l *Lexer) readString(quote rune, startLine, startCol int) (Token, error) {
	l.advance() // opening quote
	var b strings.Builder
	for {
		if l.pos >= len(l.src) {
			return Token{}, &LexError{File: l.file, Msg: "unterminated string literal", Line: startLine, Col: startCol}
		}
		ch := l.src[l.pos]
		if ch == quote {
			l.advance()
			return Token{Type: TOKEN_STRING, Lexeme: b.String(), Line: startLine, Col: startCol}, nil
		}
		if ch == '\\' {
			l.advance()
			if l.pos >= len(l.src) {
				return Token{}, &LexError{File: l.file, Msg: "unterminated escape in string", Line: startLine, Col: startCol}
			}
			esc := l.src[l.pos]
			switch esc {
			case 'n':
				b.WriteRune('\n')
			case 'r':
				b.WriteRune('\r')
			case 't':
				b.WriteRune('\t')
			case '\\':
				b.WriteRune('\\')
			case '"':
				b.WriteRune('"')
			case '\'':
				b.WriteRune('\'')
			case '0':
				b.WriteRune(0)
			default:
				return Token{}, &LexError{File: l.file, Msg: fmt.Sprintf("unknown escape sequence \\%c", esc), Line: l.line, Col: l.col}
			}
			l.advance()
			continue
		}
		b.WriteRune(ch)
		l.advance()
	}
}

func (l *Lexer) readVarRef(startLine, startCol int) (Token, error) {
	l.advance() // $
	if l.pos >= len(l.src) || !isIdentStart(l.src[l.pos]) {
		return Token{}, &LexError{File: l.file, Msg: "expected identifier after '$'", Line: startLine, Col: startCol}
	}
	var b strings.Builder
	for l.pos < len(l.src) && isIdentPart(l.src[l.pos]) {
		b.WriteRune(l.src[l.pos])
		l.advance()
	}
	name := b.String()
	if len(name) > 64 {
		return Token{}, &LexError{File: l.file, Msg: fmt.Sprintf("variable name %q exceeds 64 characters", name), Line: startLine, Col: startCol}
	}
	return Token{Type: TOKEN_VARREF, Lexeme: name, Line: startLine, Col: startCol}, nil
}

func (l *Lexer) readNumber(startLine, startCol int) (Token, error) {
	// Non-decimal integer prefixes (M12). `0x`, `0o`, `0b` followed by
	// at least one digit of the right base; `_` may appear between digits
	// as a visual separator. The lexeme stored on the token includes the
	// prefix so the parser can pick the base back out, but excludes the
	// `_` separators (parser-side ParseInt sees clean digits).
	if l.pos+1 < len(l.src) && l.src[l.pos] == '0' {
		switch l.src[l.pos+1] {
		case 'x', 'X':
			return l.readBasedInt(startLine, startCol, 16, "0x", isHexDigit)
		case 'o', 'O':
			return l.readBasedInt(startLine, startCol, 8, "0o", isOctDigit)
		case 'b', 'B':
			return l.readBasedInt(startLine, startCol, 2, "0b", isBinDigit)
		}
	}
	// Decimal int / float. `_` is accepted between digits (`1_000_000`) but
	// not as the first / last character of the integer-or-mantissa part and
	// never adjacent to the `.`.
	digits, err := l.readSeparatedDigits(startLine, startCol, unicode.IsDigit, "decimal")
	if err != nil {
		return Token{}, err
	}
	if l.pos+1 < len(l.src) && l.src[l.pos] == '.' && unicode.IsDigit(l.src[l.pos+1]) {
		l.advance() // consume the `.`
		fraction, err := l.readSeparatedDigits(startLine, startCol, unicode.IsDigit, "decimal")
		if err != nil {
			return Token{}, err
		}
		return Token{Type: TOKEN_FLOAT, Lexeme: digits + "." + fraction, Line: startLine, Col: startCol}, nil
	}
	return Token{Type: TOKEN_INT, Lexeme: digits, Line: startLine, Col: startCol}, nil
}

// readBasedInt reads a `0x...` / `0o...` / `0b...` integer literal. The
// caller has only peeked at the prefix; we advance past it here, then
// scan one or more digits of the requested base with `_` allowed
// between (but not at) digit positions. The returned lexeme carries the
// prefix back so the parser knows what base to use; underscores are
// stripped.
func (l *Lexer) readBasedInt(startLine, startCol, base int, prefix string, isDigit func(rune) bool) (Token, error) {
	l.advance() // 0
	l.advance() // x/o/b
	if l.pos >= len(l.src) || !isDigit(l.src[l.pos]) {
		return Token{}, &LexError{File: l.file, Msg: fmt.Sprintf("expected %s digit after `%s`", baseName(base), prefix), Line: startLine, Col: startCol}
	}
	digits, err := l.readSeparatedDigits(startLine, startCol, isDigit, baseName(base))
	if err != nil {
		return Token{}, err
	}
	return Token{Type: TOKEN_INT, Lexeme: prefix + digits, Line: startLine, Col: startCol}, nil
}

// readSeparatedDigits scans a run of digits with `_` allowed between
// (but not at the start or end of) the run. The returned string is the
// digit characters with the underscores stripped, so callers can hand
// it to strconv.ParseInt directly.
func (l *Lexer) readSeparatedDigits(startLine, startCol int, isDigit func(rune) bool, kind string) (string, error) {
	var b strings.Builder
	prevWasUnderscore := false
	for l.pos < len(l.src) {
		c := l.src[l.pos]
		if isDigit(c) {
			b.WriteRune(c)
			prevWasUnderscore = false
			l.advance()
			continue
		}
		if c == '_' {
			if b.Len() == 0 {
				return "", &LexError{File: l.file, Msg: fmt.Sprintf("`_` digit separator may not start a %s literal", kind), Line: startLine, Col: startCol}
			}
			if prevWasUnderscore {
				return "", &LexError{File: l.file, Msg: "consecutive `_` in numeric literal", Line: startLine, Col: startCol}
			}
			prevWasUnderscore = true
			l.advance()
			continue
		}
		break
	}
	if prevWasUnderscore {
		return "", &LexError{File: l.file, Msg: fmt.Sprintf("`_` digit separator may not end a %s literal", kind), Line: startLine, Col: startCol}
	}
	return b.String(), nil
}

func isHexDigit(r rune) bool {
	return unicode.IsDigit(r) || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')
}

func isOctDigit(r rune) bool {
	return r >= '0' && r <= '7'
}

func isBinDigit(r rune) bool {
	return r == '0' || r == '1'
}

func baseName(base int) string {
	switch base {
	case 2:
		return "binary"
	case 8:
		return "octal"
	case 16:
		return "hex"
	}
	return "decimal"
}

func (l *Lexer) readIdentifierOrKeyword(startLine, startCol int) (Token, error) {
	var b strings.Builder
	// Bare IDENTs may include `_` in the middle to support constant names
	// like MAX_RETRIES. The parser enforces per-context rules: constants
	// accept `[A-Z][A-Z_]*[A-Z]` (or single [A-Z]); variables, methods and
	// parameters reject `_` entirely.
	for l.pos < len(l.src) && isIdentPartLoose(l.src[l.pos]) {
		b.WriteRune(l.src[l.pos])
		l.advance()
	}
	name := b.String()
	if tt, ok := lookupKeyword(name); ok {
		return Token{Type: tt, Lexeme: name, Line: startLine, Col: startCol}, nil
	}
	if len(name) > 64 {
		return Token{}, &LexError{File: l.file, Msg: fmt.Sprintf("identifier %q exceeds 64 characters", name), Line: startLine, Col: startCol}
	}
	// Trailing `_` is never legal in any identifier kind.
	if name[len(name)-1] == '_' {
		return Token{}, &LexError{File: l.file, Msg: fmt.Sprintf("identifier %q may not end with `_`", name), Line: startLine, Col: startCol}
	}
	return Token{Type: TOKEN_IDENT, Lexeme: name, Line: startLine, Col: startCol}, nil
}

// isIdentStart: identifiers (variable / method / constant / parameter
// names, plus library names) must start with a letter. Digits and `_`
// are explicitly rejected as the first character.
func isIdentStart(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

// isIdentPart: variable references (`$name`) and similar contexts where
// the spec keeps names letters-only. Constant identifiers use the looser
// rule below.
func isIdentPart(r rune) bool {
	return isIdentStart(r)
}

// isIdentPartLoose: bare-IDENT continuation. Accepts `_` so the lexer can
// produce constant names like MAX_RETRIES as a single token. The trailing-
// `_` and per-context rules (uppercase-only for constants, no `_` at all
// for variables / methods / parameters) are enforced by the lexer's
// trailing-character check and by the parser at the relevant def sites.
func isIdentPartLoose(r rune) bool {
	return isIdentStart(r) || r == '_'
}
