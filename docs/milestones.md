# Jennifer - Milestones

Development is split into milestones. Each milestone produces a *working*
interpreter that runs a strictly larger subset of the language.

---

## M1 - End-to-end MVP

**Status:** done.

Smallest vertical slice that proves the pipeline (source → tokens →
preprocessed tokens → AST → result):

- Types: `int`, `string`
- `def x as int init 5;`, `$var` references, method defs (zero-arg, top
  level), `import "file.j";`, `use io;`, single-arg `printf`
- Arithmetic `+ - * / %` on ints; comments `#` and `/* */`
- Source-context caret in error messages
- Golden-file integration test and TinyGo build verified

**Exit criterion:** `./jennifer run examples/hello.j` prints `42`.

---

## M2 - Types, constants, scoping, control flow

**Status:** done.

Rounds out the "ordinary" feature set:

- New types `float`, `null`, `bool` with literals `3.14`, `null`, `true`,
  `false`
- Uninitialized `def x as T;` gives `T`'s zero value
- `def const NAME as TYPE init VALUE;` (reassignment is an error)
- Nested block scoping; inner scopes cannot redeclare visible names
- Assignment statement `$x = EXPR;`
- Comparison `< > <= >= ==`, `+` for string concat, `int`↔`float`
  promotion
- Escape parsing in `'...'` strings (previously only `"..."`)
- Control flow: `if`/`elseif`/`else`, `while`, `for`, all requiring
  `bool` conditions (no implicit truthiness)

---

## M3 - Methods with parameters and return values

**Status:** done.

- `func name(a as int, b as string) { ... }` with typed parameters,
  by-value argument passing, call-site arity + type checks
- `return;` and `return EXPR;`; recursion works
- `sprintf` and format verbs `%d %f %s %t %v %%` for both `printf` and
  `sprintf`
- The omnibus `stdlib` retired in favor of topic-based libraries; `io`
  is the first.

---

## M4 - Polish & ergonomics

**Status:** done.

- Logical operators `and`, `or`, `not` (word-based, short-circuit)
- Unary minus
- Python-3 division: `/` always returns `float`; new `div` keyword for 
  floor division (`//` is taken by line comments)
- Floats always display with a decimal (`5.0`, not `5`) so the type
  stays visible
- New libraries (all `use`-gated): [`convert`](libraries/convert.md),
  [`math`](libraries/math.md), [`strings`](libraries/strings.md)
- Interpreter gained `RegisterConst` so libraries can expose constants
  (`PI`, `E`).

---

## M5 - Interpreter improvements

**Status:** done.

- **Cross-file error sources** - errors raised inside an imported `.j`
  display the line from the imported file. See
  [technical/interpreter.md > Errors and positions](technical/interpreter.md#errors-and-positions-cross-file).
- **REPL** - `jennifer repl`, persistent globals/methods/imports across
  inputs, multi-line input via brace balancing, expression results
  printed. See [technical/cli.md > REPL](technical/cli.md#repl-cmdjenniferreplgo).
- **REPL line editor** - cursor keys, Home/End, word motions, Ctrl+W /
  Ctrl+U / Ctrl+K, in-memory history (Up/Down), Ctrl+C cancel. Non-TTY
  stdin falls back to plain line reading. See
  [technical/cli.md > Line editor](technical/cli.md#line-editor-cmdjenniferlineeditgo-cmdjenniferhistorygo).
- **Auto-loaded `core` library** - new library kind, pre-imported at
  startup; writing `use core;` is a runtime error. Contents:
  `JENNIFER_VERSION` (a `git describe`-derived build-version constant)
  and `len` (polymorphic over strings now; lists/maps in M6). `len`
  moved here from `strings`. See [libraries/core.md](libraries/core.md)
  and [technical/cli.md > Version injection](technical/cli.md#version-injection).
- **Formatter** - `jennifer fmt` re-emits canonical source per
  [user-guide/style-guide.md](user-guide/style-guide.md). Token-level walker so file imports and
  user-written parentheses survive. See
  [technical/cli.md > Formatter](technical/cli.md#formatter-cmdjenniferfmtgo).
- **Inspection subcommands** - `jennifer tokens <file>` dumps the lexer
  output; `jennifer ast <file>` dumps the preprocessed AST as JSON.
  See [technical/cli.md > Inspection](technical/cli.md#inspection-tokens-and-ast).
- **Underscore-in-constants** - constant names became `[A-Z]+(_[A-Z]+)*`,
  enabling `MAX_RETRIES` and the `JENNIFER_VERSION` rename. See
  [technical/lexer.md > Identifier rule](technical/lexer.md#identifier-rule).
- **Documentation overhaul** - `docs/technical.md` split into
  `docs/technical/<topic>.md`; `docs/lib_*.md` moved to
  `docs/libraries/`; new `docs/user-guide/style-guide.md`.

---

## M6 - Lists and maps

**Status:** done.

Two new compound types - `list` and `map` - plus the strings library
functions deferred until compound types existed.

- **Syntax**: `def xs as list of int init [1, 2, 3];`,
  `def m as map of string to int init {"a": 1};`. Index read/write
  `$xs[i]`, `$m["k"]`, chains `$g[i][j]`. Iteration via
  `for (def x in $coll) { ... }` (new keyword `in`). New tokens
  `[ ] :` and keywords `list`, `map`, `of`, `to`, `in`.
- **Semantics**: value-typed (copy on assignment and on
  function-parameter binding; no aliasing); `const` is deep
  (`$NUMS[0] = ...` is a runtime error if `NUMS` is `const`);
  out-of-bounds list reads/writes and missing map keys are positioned
  runtime errors; map iteration is insertion-order deterministic.
- **Type system**: `parser.Type` became a recursive struct
  (`Element`, `KeyType`, `ValType` `*Type` slots), so nesting like
  `list of list of int` falls out without depth cap. 3+ levels is a
  documented code smell.
- **Stdlib**: `core.len` extended to lists and maps; `core.has(m, key)`
  for membership tests; `strings.split`, `strings.chars`,
  `strings.join` finished.
- **Tooling**: formatter handles `[...]` / `{...}` per
  [user-guide/style-guide.md](user-guide/style-guide.md) (no inner padding, space after `,`/`:`,
  block-vs-map disambiguation via a small brace stack); AST JSON
  emitter handles `ListLit`, `MapLit`, `IndexExpr`, `IndexAssignStmt`,
  `ForEachStmt`.

See [user-guide/types-and-values.md > Lists and maps](user-guide/types-and-values.md#lists-and-maps) for
the user-facing tour, and
[technical/grammar.md](technical/grammar.md) /
[technical/interpreter.md](technical/interpreter.md) for the
implementation contract.

---

## M7 - printf modifiers, stdin input, comment/division swap

**Status:** done.

A breaking syntax change to free up `//` for integer division and to
allow shebangs, the long-promised format-verb modifier system, and
the first stdin-reading builtins.

- **Comments and integer division** (**BREAKING**). Line comments
  moved from `//` to `#`, freeing `//` for floor division (Python 3
  shape). `div` keyword removed. A Jennifer file can now begin with
  `#!/usr/bin/env -S jennifer run`.
- **`(s)printf` format-verb modifiers.** Each format verb except `%v`
  accepts a pipe-separated, order-independent flag list:
  `%verb[|key=value]*`. Modifiers shape *presentation* only - data
  transformations (`case=upper` on strings, markdown rendering, etc.)
  are explicitly out of scope; libraries do that work. Verbs gained:
  `pad`/`max`/`align`/`mode` (`%s`); `pad`/`fill`/`align`/`base`/
  `sign`/`group`/`sep` (`%d`); `prec`/`trim`/`sci`/`pad`/`align`/
  `sign` (`%f`); `case` (`%t`); shared `null=empty|null|literal(...)`
  across all four typed verbs. `%v` deliberately takes none.
- **Format-string breaking change.** `|` immediately after a verb
  now starts a modifier list. Pre-M7 strings with `|` as a literal
  separator (`"%d|%d"`) need either a different separator or the
  `||` escape (parallels `%%`).
- **`io` stdin input.** New builtins `readLine()`,
  `readLine(prompt)`, `eof()` - one-line-at-a-time reads with an
  explicit EOF predicate (`while (not eof()) { ... }`). Refuses
  inside the REPL since the line editor owns stdin.
- **Internals.** Builtin signature changed from
  `func(out io.Writer, args)` to `func(ctx BuiltinCtx, args)` so
  stdin and the REPL flag are plumbed symmetrically with stdout.
  Mechanical refactor across the ~30 existing builtins.

See:
- [libraries/io.md](libraries/io.md) - full modifier and input reference.
- [technical/lexer.md](technical/lexer.md) and
  [technical/grammar.md](technical/grammar.md) - the comment / division
  syntax change.
- [technical/rejected.md](technical/rejected.md) - what the modifier
  system deliberately doesn't do (data transformations, `%a`
  aggregate, `null=sql`/`null=skip`) and why the literal-pipe
  lookahead alternative was turned down.
- [technical/interpreter.md > Builtins and libraries](technical/interpreter.md#builtins-and-libraries) - the `BuiltinCtx` signature.

---

## M8 - System library namespacing

**Status:** done.

A hybrid namespace model so domain libraries can ship without
polluting the bare-name pool, plus the first real namespaced
library (`os`) so the machinery has a non-synthetic exercise.

- **Hybrid model.** Essential libraries (`io`, `convert`, `math`,
  `strings`, auto-loaded `core`) stay flat - their builtins are
  bare names. Domain libraries register through a new namespaced
  API (`RegisterNamespaced` / `RegisterNamespacedConst`) and are
  addressed by `prefix.name(...)` / `prefix.NAME`. The library's
  name doubles as the namespace prefix.
- **Qualified calls and constants.** New AST nodes
  `QualifiedCallExpr` and `QualifiedConstRefExpr`; parsed as
  `IDENT "." IDENT` (then `(` decides). Lookup is keyed by
  `(namespace, name)` and gated by `use lib;`.
- **`use NAME as ALIAS;` aliasing.** Optional `as` clause on
  `use`. Rename-not-addition: after `use bio as b;` only `b.`
  resolves, `bio.foo()` errors with a "did you mean `b`?" hint;
  the canonical name `bio` is freed for ordinary identifier use.
  Matches Python's `import foo as bar`. Aliasing is rejected for
  flat libraries (`use math as m;` errors as meaningless).
- **Namespace prefix is a reserved identifier.** After bare
  `use bio;`, `func bio() {}` errors with `shadows imported
  namespace 'bio'`. After `use bio as b;`, only `b` is reserved.
- **No migration.** The change is purely additive; all five flat
  essentials continue to work unchanged.
- **Demo library `os` (minimal slice).** First namespaced
  library: `os.platform() -> string`, `os.getEnv(name) -> string`,
  `os.JENNIFER_LF`, `os.JENNIFER_OS`. Two functions plus two
  constants - enough to exercise namespaced zero-arg calls,
  namespaced calls with arguments, namespaced constants, and
  aliasing end-to-end. Expands in M15.1.

See:
- [libraries/os.md](libraries/os.md) - the shipping demo library.
- [libraries/index.md](libraries/index.md) - flat vs namespaced
  policy and the rule for library authors.
- [user-guide/imports.md > Namespaced libraries and aliasing](user-guide/imports.md#namespaced-libraries-and-aliasing) -
  user-facing reference for `use NAME [as ALIAS];` and qualified
  calls.
- [user-guide/style-guide.md > Namespaced calls](user-guide/style-guide.md#namespaced-calls) -
  spacing convention around `.`.
- [technical/grammar.md](technical/grammar.md) - EBNF for
  `qualifiedCall` / `qualifiedConstRef` and the `use ... as ...`
  shape; AST table entries for the new nodes.
- [technical/interpreter.md > Namespaced libraries (M8)](technical/interpreter.md#namespaced-libraries-m8) -
  registration API, `nsPrefixes` / `nsAliasedAway` resolution
  tables, no-shadowing rule for namespace prefixes.

---

## M9 - Collection operations

**Status:** done.

Two new namespaced libraries cover the M6-deferred list/map
manipulation helpers, a small append sugar shortens the common
write pattern, and two follow-on breaking changes tidy up the
flat-vs-namespaced split.

- **`lists` library** (`use lists;`, namespaced). `lists.push`,
  `lists.pop`, `lists.first`, `lists.last`, `lists.head`,
  `lists.tail`, `lists.reverse`, `lists.sort`, `lists.contains`,
  `lists.concat`, `lists.slice`. Non-mutating - every function
  returns a new list. `sort` accepts numeric, string, or bool
  elements (mixed int/float promotes; other mixes error);
  comparator-based sort is deferred until methods are first-class.
- **`maps` library** (`use maps;`, namespaced). `maps.keys`,
  `maps.values`, `maps.has`, `maps.delete`, `maps.merge`. Same
  shape. `maps.delete` of a missing key errors (strict at
  boundaries, matching `$m[missing]`); `maps.merge` layers the
  second arg over the first.
- **Sugar: `$xs[] = item;`** - write-only target meaning "just past
  the end of the list". Equivalent to
  `$xs = lists.push($xs, item);`. Reads of `$xs[]` and chained
  forms (`$xs[0][]`) are parse errors; non-list targets error at
  runtime. New AST node `AppendStmt`.
- **BREAKING:** `has()` moved from `core` to `maps` as
  `maps.has(m, key)`. Bare `has(...)` callers now need
  `use maps;` and the qualified form. `has` was the only
  non-polymorphic name in core; `len` stays because it genuinely
  spans string / list / map.
- **BREAKING:** `strings` library moved from flat to namespaced.
  `upper(s)` → `strings.upper(s)`, `contains(s, sub)` →
  `strings.contains(s, sub)`, etc. across all 15 functions.
  `use strings;` itself is unchanged. The M8 library-author rule
  named exactly these collision-prone verbs (`contains`, `split`,
  `replace`, `join`); acting on it now keeps callers off the wrong
  shape before more libraries arrive. After M9 the remaining flat
  libraries are `io`, `convert`, `math`, and auto-loaded `core`.

See:
- [libraries/lists.md](libraries/lists.md) /
  [libraries/maps.md](libraries/maps.md) - function reference for
  each new library.
- [libraries/strings.md](libraries/strings.md) - now namespaced
  (M9 migration note at top).
- [libraries/index.md](libraries/index.md) - updated flat-vs-namespaced
  catalog and the library-author rule.
- [user-guide/imports.md](user-guide/imports.md) and
  [user-guide/types-and-values.md > The `$xs[]` append sugar](user-guide/types-and-values.md#the-xs-append-sugar) -
  user-facing reference.
- [technical/grammar.md](technical/grammar.md) - EBNF and AST entry
  for `AppendStmt`.

---

The next phase splits into four arcs after two architectural
prerequisites: M10 lands the namespace-first library architecture
that the rest of the standard library will be built on; Phase A
(M11-M13) finishes the language so libraries have something to
stand on; M14 closes the lexer-side gap (`fmt` losing comments
and shebangs) so the first wave of struct-using libraries can
ship with doc-comments intact; Phase B (M15.x) ships the
foundational libraries that every Jennifer program needs;
Phase C (M16.x) ships I/O libraries; Phase D (M17-M20) ships the
higher-level ecosystem (Jennifer-coded libraries, the module
system that unblocks them, crypto, a server). Phase E (WASM and
specialised domains) is the long horizon.

The library milestones use sub-numbering (M15.1, M15.2, ...) so
each library ships and is reviewed independently. This is the
first time we use sub-milestones; the practice is justified
because each library is small enough to land in a single sitting
once the language foundation is in place.

---

## M10 - Namespace-first library architecture

**Status:** done.

A pre-language-completion API-shape correction: every library is
now namespaced, with bare-name globals reserved as a narrow
`core`-only exception. Small implementation surface, large API
shape; pre-1.0 is the window for this kind of change.

- **BREAKING:** `io`, `math`, `convert` migrate to
  namespaced-only. `printf(x)` → `io.printf(x)`,
  `sqrt(x)` → `math.sqrt(x)`, etc. The "io is special, keep
  it flat" alternative was considered and rejected at kickoff
  to keep a uniform "every call carries its library name"
  rule. `strings`, `lists`, `maps`, `os` were already
  namespaced (M9/M8).
- **BREAKING:** `convert`'s four conversion callees are renamed
  to `convert.toInt`, `convert.toFloat`, `convert.toString`,
  `convert.toBool` so they don't collide with the type
  keywords (`int`, `float`, `string`, `bool`); `convert.typeOf`
  keeps its name. The `to`-prefix also reads as English
  ("convert to int") at the call site.
- **BREAKING:** file-splice keyword `import` → `include`.
  `include "x.j";` is the textual splice; the `import`
  keyword is reserved for the M17 module system and produces
  a migration-hint error today. Mixing-mistake diagnostics
  updated.
- **BREAKING for embedders:** registration API renamed.
  `Register` / `RegisterConst` → `RegisterGlobal` /
  `RegisterGlobalConst`, making their role explicit ("expose
  this name globally"). The namespaced API
  (`RegisterNamespaced` / `RegisterNamespacedConst`) keeps
  its name and is the recommended default. Per-library
  storage (`globalFnsByLib`, `globalConstsByLib`) so two
  libraries with the same global name can no longer silently
  overwrite each other at Install time; the resolution map
  is populated by `processImports` when a library activates.
- **`math` absorbs the planned non-crypto random helpers**:
  `math.rand()`, `math.randInt(lo, hi)`, `math.randSeed(n)`.
  Three functions don't justify their own library under the
  new threshold (next bullet); pseudo-random fits `math`'s
  pure-numeric charter. The crypto-grade variant still ships
  in M19 `crypto`. The originally planned M14.2 `random`
  library is removed.
- **`core` is the only library publishing bare-name globals.**
  `len` and `JENNIFER_VERSION` only - no `core.len` /
  `core.JENNIFER_VERSION` qualified form, because shipping the
  same name two ways violates stance #1. `core` is the
  auto-loaded escape hatch, and its asymmetric exposure is
  the whole point.
- **Three globals-publishing rules in `processImports`**, all
  forward-looking (inert today since `core` is the only
  globals-publishing library and can't be `use`d):
  1. Duplicate `use` of a globals-publishing library is
     rejected (`library 'X' already in scope`); REPL no-ops a
     repeat.
  2. `use X as Y;` where `X` has globals but no namespaced
     names is rejected as meaningless.
  3. Two active libraries publishing the same global name are
     rejected at the second `use`
     (`library "B" collides with already-active library "A"
     on global "VER"`). The pre-M10 flat-only-alias-meaningless
     check is removed for the general case but kept as rule 2.
- **Library-author guidance updated.** The
  `docs/libraries/index.md` "flat vs namespaced" framing is
  retired; the new policy is "every library is namespaced;
  only `core` ships globals via `RegisterGlobal`." The
  "deserves its own library" threshold is raised from M8's
  "3+" to **"5+ functions or constants"**: anything smaller
  folds into the most-related existing library. The non-crypto
  random helpers (3 functions) are the first case the new rule
  caught.

See:
- [libraries/io.md](libraries/io.md),
  [libraries/math.md](libraries/math.md),
  [libraries/convert.md](libraries/convert.md),
  [libraries/core.md](libraries/core.md) - migrated library
  references.
- [libraries/index.md](libraries/index.md) - retired
  flat-vs-namespaced framing; new library-author policy and
  5+ threshold.
- [user-guide/imports.md](user-guide/imports.md) - `use` and
  `include` keyword reference; namespaced-call and aliasing
  rules.
- [user-guide/types-and-values.md](user-guide/types-and-values.md) -
  `convert.toInt` / `convert.toFloat` example placement in the
  "explicit conversions" section.
- [technical/rejected.md](technical/rejected.md) - "Methods
  on structs" (M14.3 trigger; recorded here in M10's wake
  because M10's review touched the same call-shape question)
  and other related rejected alternatives.

No new language features land here - that's M11.

---

**Phase A: language completion (M11-M13).** These three milestones
close the biggest daily-use gaps and add the foundational types every
later library needs.

## M11 - Control-flow completion

**Status:** done.

Closes the biggest daily-use gap in the language and rounds out the
printf modifier table at the same time.

- **`break;` and `continue;`** inside `while`/`for`/`for-each`/`repeat`.
  Caught by the innermost enclosing loop only; a stray use at the top
  level or inside a method body that has no enclosing loop is a
  positioned runtime error. `continue` in a C-style `for` still runs
  the step expression before re-checking the condition (matches C/Go).
- **`repeat { } until (cond);`** post-test loop. New keywords `repeat`
  and `until`. `until` makes condition inversion explicit, matching
  the word-operator style (`and`/`or`/`not`); `do { } while ...`
  considered and rejected.
- **`exit;` and `exit EXPR;`** terminate the whole program. Bare form
  yields exit code 0; `exit EXPR;` requires an int. Distinct from
  `return` (method-scoped): an `exit` inside a deeply nested call
  skips every caller frame and every remaining top-level statement.
  Implemented as an `ExitSignal` sentinel error that the CLI catches
  and translates into the OS exit code.
- **Bundled: printf `%s|align=center`** rounds out the `%s` align
  modifier set. Rejected on every other typed verb because
  splitting padding around a numeric value breaks columnar
  alignment.
- **Bundled: printf `%a` aggregate verb** for lists and maps
  (deferred from M7; unblocked by M6's compound types and M9's
  collection libraries). Renders a list/map in a configurable shape
  so `io.printf("%a\n", $xs);` does the right thing without the
  caller building the string by hand. Modifiers:
  - `sep` (element separator, default `", "`)
  - `kv` (map key/value separator, default `": "`)
  - `open` / `close` (bracket pair, default `[`/`]` for lists,
    `{`/`}` for maps)
  - `depth=N` (max recursion depth; deeper levels collapse to
    `[...]` / `{...}`. Default unlimited; `depth=0` collapses at
    the top, useful for "size only" renderings)
  - `null=skip` (per-element null handling; omits null list
    elements and null map values. The other `null=` modes are
    rejected for `%a`.)

  Modifier values can be quoted (`%a|sep=", "`); the modifier-list
  parser was extended with a `"..."` form so values containing
  spaces / brackets / other reserved characters can be expressed.
  The escape set is the standard `\n \r \t \\ \"`.
- **Post-dot name relaxation.** Reserved words appearing as the
  name slot of a qualified call (after a `.`) read as identifiers,
  so `strings.repeat` keeps working after `repeat` is reserved as
  a loop keyword. Same rule covers `until` / `break` / `continue` /
  `exit` and any future keyword.

## M12 - Bytes and bit operators

- New primitive type `bytes` - **mutable** byte sequence with
  `list`-like semantics. Indexing yields `int` in `[0, 255]`;
  index-assign writes `$b[i] = 0xff;` (out-of-range writes are
  positioned runtime errors). Append via the same `$b[] = byte;`
  sugar M9 introduces for lists. **Value semantics** like lists:
  `$a = $b;` deep-copies, so passing bytes into a method can't
  surprise the caller. `const` is deep. `len($b)` returns the
  count. Rune-aware ops stay in `strings`; byte-level ops live
  here. (Mutable was chosen over immutable to fit real
  buffer-shaped workflows - I/O, hashing-while-streaming,
  in-place encrypt/decrypt - without forcing an allocate-new
  loop for each transformation.)
- New `bytes <-> string` codecs land in `convert` as
  `convert.bytesFromString($s, $codec)` and
  `convert.stringFromBytes($b, $codec)`. Two-argument shape
  follows `convert.toInt(v)` / `convert.toFloat(v)` (one input, one
  output) with `$codec` selecting the encoding (`"utf-8"` by
  default; further codecs ship with the `encoding` library in
  M15.4). The pair lives in `convert` because bytes ↔ string
  *is* a value transformation across kinds, which is `convert`'s
  charter; the codec-introspection helpers
  (`isAscii`, `lenBytes`, `lenRunes`, hex, base64) stay in
  `encoding` because their concern is the byte stream's shape,
  not the conversion itself.
- New operators `& | ^ ~ << >>` on `int` - promoted to syntax,
  not library calls. Library form would parallel arithmetic
  (`bitand` vs `+`) and violate "one way per thing."
- **Non-decimal integer literals**: hex `0xff` / `0xDEAD_BEEF`,
  octal `0o755`, binary `0b1010_0110`. All three lex as `int`
  (same kind, same operators); `_` accepted as a digit separator
  in any position except leading or trailing. Decimal literals
  also gain the `_` separator (`1_000_000`). Lexer-only change,
  no new runtime semantics. Lands here because byte and bit work
  without hex is the wrong default for the milestone.
- Resolves the M7-deferred stdin builtins that waited on the
  binary representation: `io.readBytes(n) -> bytes` (exact n;
  partial result at EOF, then `eof()` is true) and `io.readChars(n)
  -> string` (n runes, decoded from stdin's UTF-8). M7's `eof()`
  composes with both unchanged.

## M13 - Structs / records

- New `def struct Name { field as type, ... };` syntax (working
  name; revisited at start of M13).
- Literals: `Name{ field: expr, ... }`. Field access: `$p.field`.
  Value semantics like lists/maps - `def q as Name init $p;`
  copies. `const` is deep.
- Unblocks every library that wants to return composite data
  (file info, time values, network endpoints, http request /
  response).

---

## M14 - Lexer comment + blank-line preservation

A side-channel for comments and blank lines so `jennifer fmt`
stops dropping the file's documentation. Closes the two
M5-deferred items together because the same machinery covers
both.

- **Lexer carries comments as tokens** (`TOKEN_COMMENT_LINE`,
  `TOKEN_COMMENT_BLOCK`, `TOKEN_COMMENT_SHEBANG`). The first
  line's `#!` form is its own kind so `fmt` can re-emit it
  verbatim at the file head without re-deriving the rule.
- **Lexer carries one `TOKEN_BLANK_LINE` per run of
  newlines.** Consecutive blanks collapse to one token (matches
  the style rule "never more than one consecutive blank line").
- **Parser ignores both kinds** at statement boundaries, the
  same way it ignores whitespace today. No grammar change.
- **AST carries attachments**, not the comment tokens
  themselves. Each statement-level AST node grows two slots:
  `LeadingComments []Comment` (comments and blanks immediately
  before it) and `TrailingComment *Comment` (a same-line
  end-of-line comment if any). Attachment runs during parsing
  via a small look-around; the formatter consumes them.
- **`jennifer fmt` re-emits**: shebang at file head; leading
  comment block before its attached node with original
  blank-line spacing; trailing same-line comment on the
  statement's line. Comments inside an expression
  (`printf(/* note */ $x)`) are out of scope - they attach to
  the statement, not to mid-expression positions.
- **`jennifer ast`** gains optional `--with-comments` flag so
  the JSON dump includes the attachment slots; off by default
  to keep the existing test suite stable.
- **No language change.** Comments are still purely
  informational; the runtime never sees them. This milestone is
  pure pipeline plumbing.
- **Style guide updates.** The two "Limitations" bullets in
  `style-guide.md` (comments dropped, blank lines not
  preserved) are removed. The "Source file conventions" section
  notes that shebang, header comments, and inline `# why` notes
  all survive a `fmt` round-trip.

Lands here, between M13 (structs) and the M15.x library batch,
so the first wave of struct-using libraries can ship with
doc-comments that `fmt` will preserve.

---

**Phase B: foundational libraries (M15.x).** Small, frequently-used
libraries grouped under M15 with sub-numbering; each sub-milestone
ships one library independently.

## M15.1 - `os`

Expands the M8 demo slice (`os.platform()`, `os.getEnv(name)`,
`os.JENNIFER_LF`, `os.JENNIFER_OS` already ship there). One large
update covering process metadata, the `JENNIFER_*` constant set,
and the external-program execution surface. Depends on M13
(structs) for the result and handle types.

**Process metadata.**

- `os.args -> list of string` - command-line arguments passed to
  the running program (index 0 is the program name, by convention).
- Constants `os.JENNIFER_BUILD`, `os.JENNIFER_PLATFORM`,
  `os.JENNIFER_ARCHITECTURE` round out the `JENNIFER_*` set
  introduced in M8.

Process exit stays on the language statement `exit EXPR;` (M11),
not in `os` - see
[rejected.md > `os.exit(n)`](technical/rejected.md#osexitn).

**External-program execution.**

Two result structs, one process handle struct, four functions:

- `def struct os.Result { exitCode as int, stdout as string,
  stderr as string };` - what a command produced.
- `def struct os.Process { ... };` - opaque handle for an async
  child. Field shape (pid, internal state) settled at start of
  M15.1; users interact through the function API below, not the
  fields directly.

**Argument-list form, no shell.** Every variant takes the command
as a `list of string` (argv-style: program name plus arguments).
This avoids the shell-injection footguns of a single concatenated
command string. If a user genuinely wants shell parsing they pass
`["sh", "-c", $cmd]` explicitly - making the shell hop visible at
the call site.

- `os.run(argv) -> os.Result` - **blocking**. Runs the command to
  completion, captures stdout and stderr into the result, returns
  the exit code. The interpreter's stdin is not passed through
  (deferred; explicit stdin variant lands in a later sub-milestone
  if demand surfaces).
- `os.spawn(argv) -> os.Process` - **non-blocking**. Starts the
  command and returns immediately with a handle. Streams are
  buffered internally; the caller drains them through
  `os.wait` / `os.poll`.
- `os.wait(p as os.Process) -> os.Result` - block until `$p`
  terminates, then return its `os.Result`. Subsequent waits on
  the same handle return the same result (idempotent).
- `os.poll(p as os.Process) -> bool` - `true` once `$p` has
  terminated and an `os.wait` will return immediately. Pure
  predicate, no side effects.
- `os.kill(p as os.Process)` - request termination of `$p`
  (SIGTERM on POSIX). A subsequent `os.wait` returns the result
  the OS reports for a terminated child. Signal variants beyond
  SIGTERM are out of scope for M15.1.

Errors at the boundary (program not found, not executable,
permission denied, fork/exec failure) are positioned runtime
errors at the `os.run` / `os.spawn` site, matching the
"strict at boundaries" stance. Non-zero exit codes are **not**
errors - they're values in `os.Result.exitCode`; the caller
decides whether to branch on them.

## M15.2 - `time`

One library covering both date and time concerns through a single
zone-aware instant type, plus a separate span type for differences.
Splits into two sub-milestones because formatting and parsing carry
their own design surface (timezone names, locale-shaped output);
the core type plus arithmetic ships first so other M15 / M16
libraries can rely on it.

**One-type rationale.** Splitting `date` / `time` / `datetime`
(the Java pre-`java.time`, Python `datetime`/`date`/`time`,
JavaScript-`Date`-plus-libraries problem) front-loads
conversion code into every library that crosses the boundary.
Every program that needs a calendar date eventually needs a
time-of-day, and vice versa. Granularity (date-only,
time-of-day-only) is a property of formatting and parsing, not
of the value's type.

Unix timestamps are **not** a separate type; they're a
constructor (`time.fromUnix(n)`) and accessor
(`$t.unix() -> int`). Same shape for ISO 8601 strings and any
other wire format added in M15.2.2.

### M15.2.1 - core type, arithmetic, Unix

- `def struct time.Time { ... };` - opaque struct representing
  an instant on the wall-clock timeline with nanosecond
  precision and zone awareness. Fields are private API; users
  interact through the function set below. (Re-using the M13
  struct machinery; `time.Time` is just a struct that the
  library happens to ship and on which the library defines
  operators.)
- `def struct time.Duration { ... };` - a span. Subtracting two
  `time.Time` produces a `Duration`; adding a `Duration` to a
  `Time` produces a `Time`.
- `time.now() -> time.Time` - current instant in the local
  zone.
- `time.utc() -> time.Time` - current instant in UTC.
- `time.fromUnix(seconds as int) -> time.Time`,
  `time.fromUnixMillis(ms as int) -> time.Time`,
  `time.fromUnixNanos(ns as int) -> time.Time` - constructors
  from common Unix integer encodings.
- Accessors as plain functions:
  `time.unix($t) -> int`, `time.unixMillis($t) -> int`,
  `time.unixNanos($t) -> int`; calendar accessors
  `time.year($t)`, `time.month($t)`, `time.day($t)`,
  `time.hour($t)`, `time.minute($t)`, `time.second($t)`,
  `time.nanosecond($t)`, `time.weekday($t)`. (Jennifer has no
  methods on structs - see
  [rejected.md > Methods on structs](technical/rejected.md#methods-on-structs)
  for why; every library exposes accessors as
  `lib.name(struct)`, not `$struct.name()`.)
- Arithmetic: `time.add($t, $d) -> time.Time`,
  `time.sub($t1, $t2) -> time.Duration`,
  `time.before($a, $b) -> bool`, `time.after($a, $b) -> bool`,
  `time.equal($a, $b) -> bool`. Operator overloading is not
  in the language; these are explicit function calls.
- Duration constructors mirror the `fromUnix` shape so
  constructor and accessor never collide:
  `time.fromSeconds(n)`, `time.fromMilliseconds(n)`,
  `time.fromMinutes(n)`, `time.fromHours(n)`. Duration
  accessors return the span as the requested unit:
  `time.seconds($d)`, `time.milliseconds($d)`,
  `time.minutes($d)`, `time.hours($d)`.

### M15.2.2 - formatting, parsing, timezones

- `time.format($t, layout as string) -> string` - layout
  language settled at the start of M15.2.2 (`strftime`-style
  vs Go's reference-time style; the former wins on
  familiarity, the latter on copy-paste reliability).
- `time.parse(s as string, layout as string) -> time.Time` -
  strict parse, positioned error on mismatch.
- `time.iso(t)` / `time.fromIso(s)` - ISO 8601 round-trip;
  the common case shouldn't need a format string.
- Timezone handling:
  `time.zone(name as string) -> time.Zone`,
  `time.inZone($t, $z) -> time.Time`. Zone name uses the
  IANA database (`"Europe/Vienna"`, `"America/Los_Angeles"`).
  TinyGo footprint of the tz database evaluated honestly
  before commitment; if too heavy, ship only fixed offsets
  in M15.2.2 and defer IANA loading until embedding decisions
  settle.

## M15.3 - `hash` and `crc`

Common digests over `bytes` (MD5, SHA-1, SHA-256, CRC32, CRC64).
Pure compute, no external dependencies. Crypto-relevant primitives
that don't require key material live here; key-based crypto goes
in M19.

**One-shot API.** Whole-input digests for the common case where
the data fits in memory:

- `hash.md5($bytes) -> bytes`, `hash.sha1($bytes) -> bytes`,
  `hash.sha256($bytes) -> bytes`, `crc.crc32($bytes) -> bytes`,
  `crc.crc64($bytes) -> bytes`. Output is the raw digest as
  `bytes` - users hex/base64-encode it through `encoding` when
  they need a string representation.

**Streaming API.** For inputs larger than memory (files,
streams), one primitive per algorithm:

- `hash.streamMd5() -> hash.Stream`, `hash.streamSha256()`,
  `crc.streamCrc32()`, ... - each returns an opaque stream
  handle (struct from M13).
- `hash.update($stream, $bytes)` feeds the next chunk.
- `hash.finalize($stream) -> bytes` returns the final digest;
  the stream is consumed and further `update` calls error.
- File hashing is the documented three-line idiom: open the
  file via `fs`, read chunks, feed to a stream, finalize. The
  `hash` library does *not* ship a `hash.md5File(path)`
  convenience - that would pull `fs` into the dependency
  graph and create a parallel API for what the streaming
  primitive already does.

**No convenience wrappers** (`hash.md5String($s)`,
`hash.md5Hex($bytes)`, ...). Stance #1 "one way per thing":
strings go through `convert.bytesFromString` first, hex
encoding goes through `encoding.hex` afterwards. The composition
reads at the call site instead of multiplying out the verb names.

**Struct hashing is deferred.** Hashing a struct requires a
stable byte serialization (field order, padding, null handling)
which is its own design problem. Users serialize through the
relevant library (`json`, `csv`, future `cbor`) and hash the
resulting bytes; the `hash` library has no opinion on struct
layout.

## M15.4 - `encoding`

`encoding.isAscii`, `encoding.lenBytes`, `encoding.lenRunes`,
hex / base64 helpers, and the lossless re-encoding codecs
beyond UTF-8. Operates on the byte stream's shape (introspection
and re-encoding); the cross-kind `bytes <-> string` codec pair
itself lives in `convert` as `convert.bytesFromString` /
`convert.stringFromBytes` (shipped with the `bytes` type in
M12).

---

**Phase C: I/O libraries (M16.x).** System libraries that touch the
OS or do significant compute.

## M16.1 - `fs`

File I/O. `fs.read`, `fs.write` for `string` and `bytes`,
`fs.stat` returning a struct, directory walk. Requires M11
(bytes) and M13 (structs). Brings the M7-deferred file handles
and any non-stdin input source (`fs.open(path) -> handle`,
`handle.readLine()`, etc.) under one library.

## M16.2 - `net`

Sockets. Plain TCP and UDP first; TLS deferred to its own
sub-milestone. The first place a real `bytes` round-trip
matters.

## M16.3 - `regex`

Regular expressions over `string`. Large standalone milestone;
pure string processing, no other dependencies.

---

**Phase D: higher-level and Jennifer-coded libraries (M17-M20).**

## M17 - Module system for Jennifer-coded libraries

Real modules so Jennifer-coded libraries get namespaces, scope,
and explicit exports. Adopts "**module**" as the canonical
wording for a distributable `.j` library (matches Python, ES2015,
Rust); "bundle" deliberately not used because it overloads the
JS build-output term. The `include "x.j";` keyword (renamed in
M10) stays as the textual-splice form for inline composition
inside one source tree; library distribution moves to modules
via `import`.

- **Source tree separation.** A new top-level `modules/` directory
  holds Jennifer-coded modules; system (Go-built) libraries stay
  in `internal/lib/*/`. The split is load-bearing for distro
  packaging: system libraries are baked into the interpreter
  binary, so a Debian package only ships `/usr/bin/jennifer`;
  modules are data files that ship to a system module dir
  (`/usr/share/jennifer/modules/` or `/var/lib/jennifer/modules/`,
  decided at packaging time) and are loadable without
  recompilation. The directory exists from M17; the modules
  shipped in M18.x land directly in it.
- **`import "modules/foo.j" as foo;` syntax.** A real module
  boundary - the imported file's top-level names live behind the
  `foo.` prefix at the call site, the same shape as a namespaced
  system library. Aliasing rules match `use NAME as ALIAS;`.
- **Module resolution.** A search path with explicit precedence:
  the current source file's directory first, then any
  `-I` directories passed on the command line, then the system
  module dir. No implicit fallback to system libraries - a
  `use NAME;` always means a system library, `import "..." as ...;`
  always means a module file. Matches the explicit-prefix rule
  recorded in
  [rejected.md > Implicit `use NAME;` fallback chain](technical/rejected.md#implicit-use-name-fallback-chain-m8).
- **Module-level exports** ship with the milestone: top-level
  `def` and `func` are exported by default; a leading `private`
  marker (or similar; settled at start of M17) hides them. Same
  visibility rule top-to-bottom; no per-file export list.

## M18.x - Jennifer-coded modules

Built atop the existing system libraries. Each one ships as a
Jennifer **module** under `modules/` (the directory introduced in
M17); none of them are compiled into the interpreter binary.
Sub-milestones in priority order:

- **M18.1 - `http`** (client) - atop `net`.
- **M18.2 - `json`** - data interchange ubiquity.
- **M18.3 - `csv`** - simple, useful early.
- **M18.4 - `yaml`, `xml`, `markdown`, `pretty`** - one or more
  sub-milestones depending on scope when planned.

## M19 - `crypto`

Symmetric and asymmetric primitives, key derivation,
crypto-grade random. System library; TinyGo-safe primitives
only. Hashes already shipped in M15.3.

## M20 - `httpd`

Pure-Jennifer HTTP server atop `net`. Ships as a module under
`modules/httpd.j` (same packaging shape as the M18 modules), not
baked into the interpreter. The point where Jennifer becomes
useful for serving content.

---

**Phase E: WASM and specialised domains (M21+).** Not committed to a
timeline; recorded so the design doesn't foreclose them.

## M21 - WASM runtime embedding

Wazero or similar inside the interpreter binary. TinyGo-size
cost evaluated honestly before commitment. Without M21, no WASM
libraries.

## M22.x - WASM libraries

If M21 ships, sandboxed plugins via `use wasm:libname;`. Each
library a sub-milestone.

## M23+ - Specialised domains

Each domain its own milestone with sub-milestones as needed:

- **ML.** Vector, matrix, stats, ML primitives.
- **Bioinformatics.** Sequence alignment (Smith-Waterman,
  Needleman-Wunsch), FASTA/FASTQ parsers, molecule structures.
- **Sandbox.** Restricted-capability execution.

Ordered when demand surfaces. WASM libraries (M22.x) may cover
some of this space first.

---

## Path to 1.0.0 distribution (parallel track)

Not milestone-gated; items can be picked up between language
milestones. The earlier this track makes Jennifer visible, the
sooner breaking changes feed into a real audience before 1.0.0
locks the API.

- **CI.** GitHub Actions running `go test ./...` and
  `tinygo build` on every push and PR.
- **Tagged releases.** Prebuilt binaries on GitHub releases.
  Linux/amd64 first, then arm64; macOS and Windows after
  cross-platform support lands.
- **Debian package** (`.deb`). Hosted from a GitHub release
  artifact initially.
- **Arch.** AUR `jennifer-bin` (prebuilt) and `jennifer-git`
  (source) packages.
- **Documentation site.** GitHub Pages driven from `docs/`
  (mdBook or similar; needs a design pass).
- **Optional later:** Homebrew tap, Snap, Nix package.

---

## Long horizon (recorded, not scheduled)

- **Cross-platform support.** Today Jennifer targets Linux only.
  Windows and macOS are planned. When touching filesystem, paths,
  line endings, or process behavior, prefer portable stdlib
  helpers (`path/filepath`, not hardcoded `/`); avoid Linux-only
  assumptions.
- **McFly OS kernel integration.** Embed the Jennifer interpreter
  into **McFly OS**, an experimental OS also written in TinyGo.
  Reinforces the TinyGo-friendliness discipline: no
  `reflect`-heavy code, no goroutines in the core, no heavy
  stdlib dependencies, and no hard dependencies on a hosted
  runtime (ambient stdin, network, dynamic linking).
- **FCGI.** `use FCGI as web;` library when `net` and `httpd`
  mature. Lets Jennifer host CGI / FastCGI workloads end-to-end.
- **Inline assembler.**
- **Binary AST cache (`.jc` files).** Pre-parsed loading for big
  programs and McFly OS embedding. Its own milestone when it
  lands - file-format design, versioning, and TinyGo-safe
  serialization are enough work to merit dedicated treatment. The
  text JSON form via `jennifer ast` is the placeholder until then.
  Deferred from M5.
- **`io.lines() -> list of string`.** Slurp the whole stdin into a
  list. Additive on top of the streaming `readLine()` + `eof()`
  idiom; nice-to-have for tiny scripts, not blocking. Deferred from
  M7.
