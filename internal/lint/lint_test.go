// SPDX-License-Identifier: LGPL-3.0-only
// Copyright (C) 2026 <developer@mplx.eu>

package lint_test

import (
	"testing"

	"github.com/mplx/jennifer-lang/internal/lexer"
	"github.com/mplx/jennifer-lang/internal/lint"
	"github.com/mplx/jennifer-lang/internal/parser"
)

// lintSrc lints src with the given checks enabled and returns the findings.
// It mirrors the CLI: parse from a copy (ParseTokens strips trivia in place),
// but hand the untouched token stream to Check so suppression directives
// survive.
func lintSrc(t *testing.T, src string, enabled map[string]bool, cfg lint.Config) ([]lint.Diagnostic, error) {
	t.Helper()
	raw, err := lexer.TokenizeWithFile(src, "test.j")
	if err != nil {
		t.Fatalf("tokenize: %v", err)
	}
	parseToks := make([]lexer.Token, len(raw))
	copy(parseToks, raw)
	prog, err := parser.ParseTokens(parseToks)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return lint.Check(prog, raw, src, "test.j", enabled, cfg)
}

// only builds an enabled-set with just the given IDs.
func only(ids ...string) map[string]bool {
	m := map[string]bool{}
	for _, id := range ids {
		m[id] = true
	}
	return m
}

// countID returns how many findings carry the given ID.
func countID(diags []lint.Diagnostic, id string) int {
	n := 0
	for _, d := range diags {
		if d.ID == id {
			n++
		}
	}
	return n
}

func mustLint(t *testing.T, src string, enabled map[string]bool, cfg lint.Config) []lint.Diagnostic {
	t.Helper()
	diags, err := lintSrc(t, src, enabled, cfg)
	if err != nil {
		t.Fatalf("Check returned error: %v", err)
	}
	return diags
}

func TestUnusedLocal(t *testing.T) {
	cfg := lint.DefaultConfig()
	cases := []struct {
		name string
		src  string
		want int
	}{
		{"unused flagged", `func f() { def used as int init 1; def dead as int init 2; return $used; }`, 1},
		{"all used", `func f() { def a as int init 1; def b as int init $a; return $b; }`, 0},
		{"spawn-local not flagged", `func f() { def t as task of int init spawn { def inner as int init 5; return 0; }; return task.wait($t); }`, 0},
		{"outer used only in spawn", `func f() { def outer as int init 5; def t as task of int init spawn { return $outer; }; return task.wait($t); }`, 0},
		{"global not flagged", `def top as int init 1;`, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			diags := mustLint(t, c.src, only("L001"), cfg)
			if got := countID(diags, "L001"); got != c.want {
				t.Fatalf("L001 count = %d, want %d (%v)", got, c.want, diags)
			}
		})
	}
}

func TestDeadCode(t *testing.T) {
	diags := mustLint(t, `func f() { return 1; def x as int init 2; }`, only("L002"), lint.DefaultConfig())
	if countID(diags, "L002") != 1 {
		t.Fatalf("expected one L002, got %v", diags)
	}
	clean := mustLint(t, `func f() { def x as int init 2; return $x; }`, only("L002"), lint.DefaultConfig())
	if len(clean) != 0 {
		t.Fatalf("expected no L002, got %v", clean)
	}
}

func TestEmptyCatch(t *testing.T) {
	diags := mustLint(t, `func f() { try { risky(); } catch (e) { } }`, only("L003"), lint.DefaultConfig())
	if countID(diags, "L003") != 1 {
		t.Fatalf("expected one L003, got %v", diags)
	}
	clean := mustLint(t, `func f() { try { risky(); } catch (e) { handle($e); } }`, only("L003"), lint.DefaultConfig())
	if len(clean) != 0 {
		t.Fatalf("expected no L003, got %v", clean)
	}
}

func TestThrowNonError(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want int
	}{
		{"string throw", `func f() { throw "boom"; }`, 1},
		{"error literal", `func f() { throw Error{kind: "x", message: "m", file: "f", line: 1, col: 1}; }`, 0},
		{"catch var rethrow", `func f() { try { risky(); } catch (e) { throw $e; } }`, 0},
		{"int throw", `func f() { throw 5; }`, 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			diags := mustLint(t, c.src, only("L004"), lint.DefaultConfig())
			if got := countID(diags, "L004"); got != c.want {
				t.Fatalf("L004 count = %d, want %d (%v)", got, c.want, diags)
			}
		})
	}
}

func TestMethodTooLong(t *testing.T) {
	cfg := lint.DefaultConfig()
	cfg.MethodMaxStmts = 2
	src := `func f() { def a as int init 1; def b as int init 2; def c as int init 3; return $a; }`
	diags := mustLint(t, src, only("L005"), cfg)
	if countID(diags, "L005") != 1 {
		t.Fatalf("expected one L005, got %v", diags)
	}
}

func TestNestingTooDeep(t *testing.T) {
	cfg := lint.DefaultConfig()
	cfg.MaxNesting = 1
	src := `func f() { if (true) { if (true) { return 1; } } }`
	diags := mustLint(t, src, only("L006"), cfg)
	if countID(diags, "L006") != 1 {
		t.Fatalf("expected one L006 (shallowest violating block), got %v", diags)
	}
}

func TestConstantCondition(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want int
	}{
		{"if true", `func f() { if (true) { return 1; } }`, 1},
		{"self compare", `func f(x as int) { if ($x == $x) { return 1; } }`, 1},
		{"while true no escape", `func f() { while (true) { def a as int init 1; } }`, 1},
		{"while true with break", `func f() { while (true) { break; } }`, 0},
		{"while true with return", `func f() { while (true) { return 1; } }`, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			diags := mustLint(t, c.src, only("L007"), lint.DefaultConfig())
			if got := countID(diags, "L007"); got != c.want {
				t.Fatalf("L007 count = %d, want %d (%v)", got, c.want, diags)
			}
		})
	}
}

func TestRemovedApi(t *testing.T) {
	diags := mustLint(t, `use core;`, only("L009"), lint.DefaultConfig())
	if countID(diags, "L009") != 1 {
		t.Fatalf("expected one L009, got %v", diags)
	}
	clean := mustLint(t, `use io;`, only("L009"), lint.DefaultConfig())
	if len(clean) != 0 {
		t.Fatalf("expected no L009 for a live library, got %v", clean)
	}
}

func TestLineTooLong(t *testing.T) {
	cfg := lint.DefaultConfig()
	cfg.MaxLineLength = 20
	src := "def x as int init 123456789;\ndef y as int;\n"
	diags := mustLint(t, src, only("L010"), cfg)
	if countID(diags, "L010") != 1 {
		t.Fatalf("expected one L010 (only the long line), got %v", diags)
	}
	if diags[0].Line != 1 {
		t.Fatalf("L010 line = %d, want 1", diags[0].Line)
	}
}

func TestSuppression(t *testing.T) {
	t.Run("line directive", func(t *testing.T) {
		src := "func f() { def dead as int init 2;   # lint-disable: L001\nreturn 0; }"
		diags := mustLint(t, src, only("L001"), lint.DefaultConfig())
		if len(diags) != 0 {
			t.Fatalf("expected suppressed, got %v", diags)
		}
	})
	t.Run("file directive", func(t *testing.T) {
		src := "# lint-disable-file: L001\nfunc f() { def dead as int init 2; return 0; }"
		diags := mustLint(t, src, only("L001"), lint.DefaultConfig())
		if len(diags) != 0 {
			t.Fatalf("expected file-suppressed, got %v", diags)
		}
	})
	t.Run("wrong id not suppressed", func(t *testing.T) {
		src := "func f() { def dead as int init 2;   # lint-disable: L002\nreturn 0; }"
		diags := mustLint(t, src, only("L001"), lint.DefaultConfig())
		if countID(diags, "L001") != 1 {
			t.Fatalf("expected L001 still reported, got %v", diags)
		}
	})
	t.Run("unknown id errors", func(t *testing.T) {
		src := "func f() { def dead as int init 2;   # lint-disable: L999\nreturn 0; }"
		if _, err := lintSrc(t, src, only("L001"), lint.DefaultConfig()); err == nil {
			t.Fatal("expected an error for an unknown suppression ID")
		}
	})
	t.Run("empty directive errors", func(t *testing.T) {
		src := "func f() { def dead as int init 2;   # lint-disable:\nreturn 0; }"
		if _, err := lintSrc(t, src, only("L001"), lint.DefaultConfig()); err == nil {
			t.Fatal("expected an error for a directive naming no IDs")
		}
	})
}

func TestResolveSelection(t *testing.T) {
	t.Run("default all", func(t *testing.T) {
		set, err := lint.ResolveSelection("", false, "", false)
		if err != nil {
			t.Fatal(err)
		}
		if len(set) != len(lint.KnownIDs()) {
			t.Fatalf("default should enable all %d checks, got %d", len(lint.KnownIDs()), len(set))
		}
	})
	t.Run("exclude", func(t *testing.T) {
		set, err := lint.ResolveSelection("!L005,!L006", true, "", false)
		if err != nil {
			t.Fatal(err)
		}
		if !set["L001"] || set["L005"] || set["L006"] {
			t.Fatalf("exclude spec wrong: %v", set)
		}
	})
	t.Run("include only", func(t *testing.T) {
		set, err := lint.ResolveSelection("L001,L002", true, "", false)
		if err != nil {
			t.Fatal(err)
		}
		if !set["L001"] || !set["L002"] || set["L003"] {
			t.Fatalf("include spec wrong: %v", set)
		}
	})
	t.Run("mixed direction errors", func(t *testing.T) {
		if _, err := lint.ResolveSelection("L001,!L002", true, "", false); err == nil {
			t.Fatal("expected error mixing include and exclude")
		}
	})
	t.Run("unknown id errors", func(t *testing.T) {
		if _, err := lint.ResolveSelection("L999", true, "", false); err == nil {
			t.Fatal("expected error for unknown ID")
		}
	})
	t.Run("dotfile used when no flag", func(t *testing.T) {
		set, err := lint.ResolveSelection("", false, "L001  # only this\n", true)
		if err != nil {
			t.Fatal(err)
		}
		if !set["L001"] || set["L002"] {
			t.Fatalf("dotfile spec wrong: %v", set)
		}
	})
}

func TestKnownIDs(t *testing.T) {
	if n := len(lint.KnownIDs()); n != 10 {
		t.Fatalf("expected 10 known IDs, got %d", n)
	}
	if len(lint.Catalog()) != 10 {
		t.Fatalf("catalog should list all 10 checks")
	}
}
