// SPDX-License-Identifier: LGPL-3.0-only
// Copyright (C) 2026 <developer@mplx.eu>

package module

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestClassify(t *testing.T) {
	cases := []struct {
		path string
		want Kind
	}{
		{"./f.j", Local},
		{"../f.j", Local},
		{"../../a/b.j", Local},
		{"/abs/f.j", Absolute},
		{"f.j", Module},
		{"sub/f.j", Module},
		{"a/b/c.j", Module},
	}
	for _, c := range cases {
		got, err := Classify(c.path)
		if err != nil {
			t.Errorf("Classify(%q): unexpected error %v", c.path, err)
			continue
		}
		if got != c.want {
			t.Errorf("Classify(%q) = %v, want %v", c.path, got, c.want)
		}
	}
}

func TestClassifyRejects(t *testing.T) {
	cases := []struct {
		path, want string
	}{
		{"", "empty"},
		{`sub\f.j`, "must use '/'"},
		{`.\f.j`, "must use '/'"},
		{"f.txt", "must end in '.j'"},
		{"nodotj", "must end in '.j'"},
	}
	for _, c := range cases {
		_, err := Classify(c.path)
		if err == nil {
			t.Errorf("Classify(%q): expected an error", c.path)
			continue
		}
		if !strings.Contains(err.Error(), c.want) {
			t.Errorf("Classify(%q): error %q missing %q", c.path, err.Error(), c.want)
		}
	}
}

func TestResolveLocalAndAbsolute(t *testing.T) {
	dir := t.TempDir()
	importing := filepath.Join(dir, "pkg")

	// Local: relative to the importing file's directory, canonicalized.
	got, err := Resolve("./util.j", importing, nil)
	if err != nil {
		t.Fatalf("local: %v", err)
	}
	if want := filepath.Join(importing, "util.j"); got != want {
		t.Errorf("local = %q, want %q", got, want)
	}
	// Local with ..: navigation within the tree is allowed.
	got, _ = Resolve("../shared/x.j", importing, nil)
	if want := filepath.Join(dir, "shared", "x.j"); got != want {
		t.Errorf("local .. = %q, want %q", got, want)
	}
	// Absolute: exactly that file, ignoring importingDir and searchDirs.
	abs := filepath.Join(dir, "opt", "m.j")
	got, err = Resolve(abs, importing, []string{"/should/be/ignored"})
	if err != nil {
		t.Fatalf("absolute: %v", err)
	}
	if got != abs {
		t.Errorf("absolute = %q, want %q", got, abs)
	}
}

func TestResolveModuleSearchPath(t *testing.T) {
	root := t.TempDir()
	sysdir := filepath.Join(root, "sys")
	idir := filepath.Join(root, "extra")
	mustMkdir(t, sysdir)
	mustMkdir(t, idir)
	mustWrite(t, filepath.Join(sysdir, "core.j"), "")
	mustWrite(t, filepath.Join(idir, "helper.j"), "")

	// Found in the system dir.
	got, err := Resolve("core.j", "/irrelevant", []string{sysdir, idir})
	if err != nil {
		t.Fatalf("core.j: %v", err)
	}
	if want := filepath.Join(sysdir, "core.j"); got != want {
		t.Errorf("core.j = %q, want %q", got, want)
	}
	// Found only in the -I dir (a -I dir adds names).
	got, err = Resolve("helper.j", "/irrelevant", []string{sysdir, idir})
	if err != nil {
		t.Fatalf("helper.j: %v", err)
	}
	if want := filepath.Join(idir, "helper.j"); got != want {
		t.Errorf("helper.j = %q, want %q", got, want)
	}
}

func TestResolveModuleDuplicateIsError(t *testing.T) {
	root := t.TempDir()
	a := filepath.Join(root, "a")
	b := filepath.Join(root, "b")
	mustMkdir(t, a)
	mustMkdir(t, b)
	mustWrite(t, filepath.Join(a, "dup.j"), "")
	mustWrite(t, filepath.Join(b, "dup.j"), "")

	_, err := Resolve("dup.j", "/x", []string{a, b})
	if err == nil {
		t.Fatal("expected an ambiguity error for a duplicate module name")
	}
	if !strings.Contains(err.Error(), "ambiguous") {
		t.Errorf("error %q should mention ambiguity", err.Error())
	}
}

func TestResolveModuleNotFound(t *testing.T) {
	if _, err := Resolve("missing.j", "/x", []string{t.TempDir()}); err == nil {
		t.Fatal("expected a not-found error")
	} else if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error %q should mention not found", err.Error())
	}
}

func TestResolveRejectsBadPath(t *testing.T) {
	if _, err := Resolve(`sub\x.j`, "/x", nil); err == nil {
		t.Error("expected backslash rejection to propagate through Resolve")
	}
}

func mustMkdir(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
