// SPDX-License-Identifier: LGPL-3.0-only
// Copyright (C) 2026 <developer@mplx.eu>

package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/mplx/jennifer-lang/internal/lexer"
	"github.com/mplx/jennifer-lang/internal/parser"
)

// TestAstJSONIsValid covers the AST JSON emitter end to end: parse a
// representative program, walk it through emitNode, and assert the
// output is parseable JSON. encoding/json is used in the *test* only -
// the emitter itself stays hand-rolled for TinyGo.
func TestAstJSONIsValid(t *testing.T) {
	cases := []string{
		`use io; printf("hi\n");`,
		`def x as int init 21; printf($x + $x);`,
		`func fact(n as int) { if ($n == 0) { return 1; } return $n * fact($n - 1); }`,
		`def const MAX_RETRIES as int init 3; printf(MAX_RETRIES);`,
		`for (def i as int init 0; $i < 3; $i = $i + 1) { printf($i); }`,
	}
	for _, src := range cases {
		prog, err := parser.Parse(src)
		if err != nil {
			t.Fatalf("parse %q: %v", src, err)
		}
		var b strings.Builder
		emitNode(&b, prog, 0)
		var v any
		if err := json.Unmarshal([]byte(b.String()), &v); err != nil {
			t.Errorf("invalid JSON for %q: %v\noutput:\n%s", src, err, b.String())
		}
	}
}

// TestAstJSONShape checks a few key fields on a known-shape program so a
// future refactor that drops or renames a field is caught here.
func TestAstJSONShape(t *testing.T) {
	prog, err := parser.Parse(`use io; def x as int init 42; printf($x);`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	var b strings.Builder
	emitNode(&b, prog, 0)
	out := b.String()

	mustContain := []string{
		`"type": "Program"`,
		`"type": "ImportStmt"`, `"name": "io"`,
		`"type": "DefineStmt"`, `"varName": "x"`, `"varType": "int"`,
		`"type": "IntLit"`, `"value": 42`,
		`"type": "ExprStmt"`,
		`"type": "CallExpr"`, `"callee": "printf"`,
		`"type": "VarExpr"`, `"name": "x"`,
	}
	for _, want := range mustContain {
		if !strings.Contains(out, want) {
			t.Errorf("AST JSON missing %q\nfull output:\n%s", want, out)
		}
	}
}

// TestTokenDumpCoversAllKinds is a thin sanity check on the lexer's
// token-type names: every kind we expect to see in real programs has a
// String() that doesn't fall through to the "TokenType(N)" fallback.
func TestTokenDumpCoversAllKinds(t *testing.T) {
	src := `use io; def const MAX as int init 1; func f() { if (true) { return; } } $x = -1.5; printf("hi");`
	tokens, err := lexer.TokenizeWithFile(src, "<test>")
	if err != nil {
		t.Fatalf("lex: %v", err)
	}
	for _, tok := range tokens {
		name := tok.Type.String()
		if strings.HasPrefix(name, "TokenType(") {
			t.Errorf("token %v has no friendly String()", tok)
		}
	}
}
