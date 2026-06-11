// SPDX-License-Identifier: LGPL-3.0-only
// Copyright (C) 2026 <developer@mplx.eu>

package lexer

import "fmt"

type TokenType int

const (
	TOKEN_EOF TokenType = iota
	TOKEN_ILLEGAL

	// Literals
	TOKEN_INT
	TOKEN_FLOAT
	TOKEN_STRING
	TOKEN_IDENT  // method names like app, printf, stdlib
	TOKEN_VARREF // $name

	// Keywords
	TOKEN_DEFINE // `def` keyword; introduces a variable or constant
	TOKEN_FUNC   // `func` keyword; introduces a method
	TOKEN_AS
	TOKEN_INIT
	TOKEN_CONST
	TOKEN_INCLUDE // textual file splice: `include "name.j";` (M10+)
	TOKEN_IMPORT  // reserved word, no live syntax (M10+ moved file splice to `include`; reserved for the M17 module system)
	TOKEN_USE     // library import: `use io;`
	TOKEN_RETURN
	TOKEN_IF
	TOKEN_ELSEIF
	TOKEN_ELSE
	TOKEN_WHILE
	TOKEN_FOR
	TOKEN_TRUE
	TOKEN_FALSE
	TOKEN_NULL
	TOKEN_AND
	TOKEN_OR
	TOKEN_NOT
	TOKEN_DIV         // `//` operator: floor (integer) division; `/` is true division (Python 3 style)
	TOKEN_INT_TYPE    // the word "int" used as a type
	TOKEN_FLOAT_TYPE  // the word "float" used as a type
	TOKEN_STRING_TYPE // the word "string" used as a type
	TOKEN_BOOL_TYPE   // the word "bool" used as a type

	// Compound-type keywords (M6)
	TOKEN_LIST // the word "list" used as a type
	TOKEN_MAP  // the word "map" used as a type
	TOKEN_OF   // "of" - element-type separator: `list of int`, `map of K to V`
	TOKEN_TO   // "to" - K/V separator inside `map of K to V`
	TOKEN_IN   // "in" - for-each iterator: `for (def x in $coll)`

	// Punctuation
	TOKEN_LBRACE   // {
	TOKEN_RBRACE   // }
	TOKEN_LPAREN   // (
	TOKEN_RPAREN   // )
	TOKEN_LBRACKET // [ (M6: list literals + index expressions)
	TOKEN_RBRACKET // ] (M6)
	TOKEN_SEMI     // ;
	TOKEN_COMMA    // ,
	TOKEN_COLON    // : (M6: map literal key-value separator)
	TOKEN_ASSIGN   // =
	TOKEN_DOT      // . (reserved; future namespacing / field access)

	// Arithmetic operators
	TOKEN_PLUS    // +
	TOKEN_MINUS   // -
	TOKEN_STAR    // *
	TOKEN_SLASH   // /
	TOKEN_PERCENT // %

	// Comparison operators
	TOKEN_LT // <
	TOKEN_GT // >
	TOKEN_LE // <=
	TOKEN_GE // >=
	TOKEN_EQ // ==
)

var tokenNames = map[TokenType]string{
	TOKEN_EOF:         "EOF",
	TOKEN_ILLEGAL:     "ILLEGAL",
	TOKEN_INT:         "INT",
	TOKEN_FLOAT:       "FLOAT",
	TOKEN_STRING:      "STRING",
	TOKEN_IDENT:       "IDENT",
	TOKEN_VARREF:      "VARREF",
	TOKEN_DEFINE:      "DEF",
	TOKEN_FUNC:        "FUNC",
	TOKEN_AS:          "AS",
	TOKEN_INIT:        "INIT",
	TOKEN_CONST:       "CONST",
	TOKEN_INCLUDE:     "INCLUDE",
	TOKEN_IMPORT:      "IMPORT",
	TOKEN_USE:         "USE",
	TOKEN_RETURN:      "RETURN",
	TOKEN_IF:          "IF",
	TOKEN_ELSEIF:      "ELSEIF",
	TOKEN_ELSE:        "ELSE",
	TOKEN_WHILE:       "WHILE",
	TOKEN_FOR:         "FOR",
	TOKEN_TRUE:        "TRUE",
	TOKEN_FALSE:       "FALSE",
	TOKEN_NULL:        "NULL",
	TOKEN_AND:         "AND",
	TOKEN_OR:          "OR",
	TOKEN_NOT:         "NOT",
	TOKEN_DIV:         "DIV",
	TOKEN_INT_TYPE:    "INT_TYPE",
	TOKEN_FLOAT_TYPE:  "FLOAT_TYPE",
	TOKEN_STRING_TYPE: "STRING_TYPE",
	TOKEN_BOOL_TYPE:   "BOOL_TYPE",
	TOKEN_LIST:        "LIST",
	TOKEN_MAP:         "MAP",
	TOKEN_OF:          "OF",
	TOKEN_TO:          "TO",
	TOKEN_IN:          "IN",
	TOKEN_LBRACE:      "LBRACE",
	TOKEN_RBRACE:      "RBRACE",
	TOKEN_LPAREN:      "LPAREN",
	TOKEN_RPAREN:      "RPAREN",
	TOKEN_LBRACKET:    "LBRACKET",
	TOKEN_RBRACKET:    "RBRACKET",
	TOKEN_SEMI:        "SEMI",
	TOKEN_COMMA:       "COMMA",
	TOKEN_COLON:       "COLON",
	TOKEN_ASSIGN:      "ASSIGN",
	TOKEN_DOT:         "DOT",
	TOKEN_PLUS:        "PLUS",
	TOKEN_MINUS:       "MINUS",
	TOKEN_STAR:        "STAR",
	TOKEN_SLASH:       "SLASH",
	TOKEN_PERCENT:     "PERCENT",
	TOKEN_LT:          "LT",
	TOKEN_GT:          "GT",
	TOKEN_LE:          "LE",
	TOKEN_GE:          "GE",
	TOKEN_EQ:          "EQ",
}

func (t TokenType) String() string {
	if name, ok := tokenNames[t]; ok {
		return name
	}
	return fmt.Sprintf("TokenType(%d)", int(t))
}

// Token is one lexeme produced by the scanner.
// Lexeme holds the literal text for identifiers, numbers, and unprocessed strings.
// For TOKEN_STRING, Lexeme holds the already-escape-processed value (no surrounding quotes).
// For TOKEN_VARREF, Lexeme holds the variable name without the leading "$".
// File is the source filename the token came from (empty if unknown / from a string).
type Token struct {
	Type   TokenType
	Lexeme string
	Line   int
	Col    int
	File   string
}

func (t Token) String() string {
	if t.File != "" {
		return fmt.Sprintf("%s(%q) @%s:%d:%d", t.Type, t.Lexeme, t.File, t.Line, t.Col)
	}
	return fmt.Sprintf("%s(%q) @%d:%d", t.Type, t.Lexeme, t.Line, t.Col)
}

var keywords = map[string]TokenType{
	"def":    TOKEN_DEFINE,
	"func":   TOKEN_FUNC,
	"as":     TOKEN_AS,
	"init":   TOKEN_INIT,
	"const":  TOKEN_CONST,
	"include": TOKEN_INCLUDE,
	"import":  TOKEN_IMPORT,
	"use":     TOKEN_USE,
	"return": TOKEN_RETURN,
	"if":     TOKEN_IF,
	"elseif": TOKEN_ELSEIF,
	"else":   TOKEN_ELSE,
	"while":  TOKEN_WHILE,
	"for":    TOKEN_FOR,
	"true":   TOKEN_TRUE,
	"false":  TOKEN_FALSE,
	"null":   TOKEN_NULL,
	"and":    TOKEN_AND,
	"or":     TOKEN_OR,
	"not":    TOKEN_NOT,
	"int":    TOKEN_INT_TYPE,
	"float":  TOKEN_FLOAT_TYPE,
	"string": TOKEN_STRING_TYPE,
	"bool":   TOKEN_BOOL_TYPE,
	"list":   TOKEN_LIST,
	"map":    TOKEN_MAP,
	"of":     TOKEN_OF,
	"to":     TOKEN_TO,
	"in":     TOKEN_IN,
}

func lookupKeyword(ident string) (TokenType, bool) {
	tt, ok := keywords[ident]
	return tt, ok
}
