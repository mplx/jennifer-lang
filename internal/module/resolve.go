// SPDX-License-Identifier: LGPL-3.0-only
// Copyright (C) 2026 <developer@mplx.eu>

// Package module implements the Jennifer module system's resolution layer:
// turning an `import "..."` path string into a concrete file on disk, and
// resolving the system module directory from its CLI / env / compile-time
// layers. It is the resolution half of the module system; the `import`
// statement, loader, and per-module scope that consume it are separate.
//
// Import paths are OS-independent logical strings, always `/`-separated
// (like a URL or the `.j` string literal). A `\` in an import path is a
// syntax error, never a Windows separator. The shape is classified on the
// logical string; only then is the file located with `path/filepath`, so
// host separators and drive letters are the stdlib's concern at resolve
// time, not the grammar's.
package module

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Kind classifies an import path by its leading token.
type Kind int

const (
	// Local: "./f.j" or "../f.j" - relative to the importing file's
	// directory; the search path is not consulted.
	Local Kind = iota
	// Absolute: an absolute filesystem path ("/abs/f.j"); the search path
	// is not consulted.
	Absolute
	// Module: a bare path ("f.j", "sub/f.j") - resolved by walking the
	// module search path; the importing file's directory is never used.
	Module
)

// Classify determines the shape of a logical import path, or returns an
// error for a malformed one (empty, backslash-bearing, or not ending in
// `.j`). It does not touch the filesystem.
func Classify(importPath string) (Kind, error) {
	if importPath == "" {
		return 0, fmt.Errorf("empty import path")
	}
	if strings.ContainsRune(importPath, '\\') {
		return 0, fmt.Errorf("import path %q must use '/' separators, not '\\' (paths are OS-independent)", importPath)
	}
	if !strings.HasSuffix(importPath, ".j") {
		return 0, fmt.Errorf("import path %q must end in '.j'", importPath)
	}
	switch {
	case strings.HasPrefix(importPath, "./") || strings.HasPrefix(importPath, "../"):
		return Local, nil
	case filepath.IsAbs(filepath.FromSlash(importPath)):
		return Absolute, nil
	default:
		return Module, nil
	}
}

// Resolve turns a logical import path into a canonical absolute file path.
//
//   - importingDir is the directory of the file that contains the import,
//     used only for Local imports.
//   - searchDirs is the ordered module search path (system module dir
//     first, then each `-I DIR`), used only for Module imports.
//
// A Module import that resolves in more than one search dir is a hard
// error (ambiguous - a `-I` dir may add names but never silently shadow a
// system module or another `-I`); one that resolves nowhere is a
// not-found error naming the search path. The returned path is cleaned and
// absolute (canonical), the key the loader uses for run-once and cycle
// detection.
func Resolve(importPath, importingDir string, searchDirs []string) (string, error) {
	kind, err := Classify(importPath)
	if err != nil {
		return "", err
	}
	native := filepath.FromSlash(importPath)
	switch kind {
	case Local:
		return canonical(filepath.Join(importingDir, native))
	case Absolute:
		return canonical(native)
	default: // Module
		return resolveModule(importPath, native, searchDirs)
	}
}

// resolveModule walks the search path, requiring exactly one match.
func resolveModule(importPath, native string, searchDirs []string) (string, error) {
	var matches []string
	for _, dir := range searchDirs {
		cand := filepath.Join(dir, native)
		if fi, err := os.Stat(cand); err == nil && !fi.IsDir() {
			c, err := canonical(cand)
			if err != nil {
				return "", err
			}
			matches = append(matches, c)
		}
	}
	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		if len(searchDirs) == 0 {
			return "", fmt.Errorf("module %q not found: the module search path is empty", importPath)
		}
		return "", fmt.Errorf("module %q not found on the search path %v", importPath, searchDirs)
	default:
		return "", fmt.Errorf("module %q is ambiguous: found in multiple search dirs: %s", importPath, strings.Join(matches, ", "))
	}
}

// canonical returns the cleaned absolute form of a path. It does not
// require the path to exist (a Local / Absolute import to a missing file
// is the loader's not-found error, positioned at the import).
func canonical(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("cannot resolve %q: %v", path, err)
	}
	return filepath.Clean(abs), nil
}
