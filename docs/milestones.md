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

The flat-namespace model is fine for the essential libraries today
(`io`, `convert`, `math`, `strings`, plus the auto-loaded `core`) -
they're small, names are carefully chosen, collisions are unlikely. It will not scale to domain
libraries like `regex`, `net`, `bio`, or `crypto`, where the chance of
two libraries shipping a `len` / `parse` / `encode` is high. M8 settles
the policy before the first domain library lands.

**Decision (to confirm at start of M8):** hybrid model. Essential
libraries stay flat for ergonomics; domain libraries are prefixed
through a namespace tag set at registration time. Optional aliasing
lets the user shorten a namespace.

**Concrete plan:**

- **Library declares a namespace at install time.**
  ```go
  // internal/lib/io/iolib.go - essential, no namespace
  in.Register("io", "printf", printf)
  // internal/lib/bio/biolib.go - domain library
  in.RegisterNamespaced("bio", "translate", translate)
  ```
  Internally: extend `builtinEntry` with an optional `Namespace string`.
  Empty namespace = flat lookup (current behavior).
- **Syntax: qualified calls via `.`.**
  The lexer already reserves `TOKEN_DOT`. Add a grammar production:
  ```ebnf
  qualifiedCall = IDENT "." IDENT "(" [ args ] ")" ;
  ```
  `primary` accepts `qualifiedCall` alongside the existing `call`. A
  bare `IDENT` immediately followed by `.` and another `IDENT` and `(`
  is parsed as a qualified call; otherwise the existing call /
  constant-ref disambiguation applies.
- **New AST node:** `QualifiedCallExpr { Namespace, Callee, Args }`.
- **Lookup rule:** at the call site, the interpreter resolves
  `bio.translate(...)` as "find a builtin whose `Namespace == "bio"`
  and whose `Name == "translate"`, registered under a library the
  program has `use`d." A bare `translate(...)` finds only the
  flat-registered builtins (and user-defined methods).
- **Constants get the same treatment.** A namespaced library exposing
  a constant uses `bio.STOPS`; flat-registered constants like `PI`
  stay bare.
- **`use bio;` is unchanged at the source level** - the user still
  imports by library name. The namespace is whatever the library
  declared, not the library name necessarily (usually they'll match,
  but a library called `crypto_primitives` could expose itself as
  `crypto.` to be brief).
- **Aliasing is a rename, not an addition.**
  ```jennifer
  use bio as b;
  printf("%d\n", b.translateLen($seq));
  bio.translateLen($seq);   # error: unknown namespace `bio`
  ```
  After `use bio as b;`, only `b.` resolves. `bio.` errors with a
  "did you mean `b`?" hint. Matches Python's `import foo as bar`
  shadowing of `foo`; matches Jennifer's "one way per thing" stance.
  Adds an `AsName` to `ImportStmt`.
- **Namespace name as a reserved identifier.** A bare `use bio;`
  reserves `bio` as a namespace-prefix identifier - `func bio() {}`
  is then rejected with `shadows imported namespace 'bio'`. With
  `use bio as b;`, only `b` is reserved, freeing `bio` as a regular
  method name. Style guide will note "don't reuse a library's
  canonical name even when aliased - it's confusing"; soft rule,
  not enforced.
- **Migration of existing libraries:** none. All five essentials stay
  flat. The change is additive; existing source keeps working
  unchanged.
- **No-shadowing rule:** scoped to a namespace. `bio.len` does NOT
  collide with `strings.len`. A user method named `translate` collides
  with `bio.translate` only if it's referenced bare; it does not
  conflict with the qualified form. Document the resolution rule
  carefully in `docs/technical/interpreter.md`.

**Demo library: `os` minimal slice.** Alongside the namespace
machinery, M8 ships the first real namespaced library so a Jennifer
program can exercise the feature without test-only registrations.
Scope is intentionally tiny:

- `os.platform() -> string` - `"linux"` today. Zero-arg
  namespaced call.
- `os.getEnv(name) -> string` - read an environment variable,
  empty string when unset. Exercises a namespaced call with an
  argument (parser path `IDENT.IDENT(STRING)`).
- `os.JENNIFER_LF` - `"\n"` today (becomes `"\r\n"` on Windows
  when cross-platform support lands).
- `os.JENNIFER_OS` - `"linux"`.

Two functions and two constants. Exercises namespaced zero-arg
calls, namespaced calls with arguments, namespaced constants, and
`use os as o;` aliasing in one place. The full `os` library
expands in M13.1 (args, exit code, the rest of the `JENNIFER_*`
constants).

**Test strategy.** The shipping `os` slice handles end-to-end
coverage; parser, interpreter, and unit tests additionally register
**synthetic** namespaced builtins inline through the public Go API
(`in.RegisterNamespaced("bio", "translate", fn)`) so namespaces other
than `os` are exercised without requiring more shipping code. The
parser sees a qualified call as `bio.translate(...)` regardless of
whether the namespace points at real code, so this covers the full
pipeline without a stub package on disk. Concretely:

- **Parser:** `QualifiedCallExpr` parses `IDENT.IDENT(...)`; table-driven
  cases include qualified constant refs (`bio.STOPS`) and rejection of
  qualified-assign (`bio.x = 1` is a parse error).
- **Interpreter:** register synthetic namespaced builtins, then
  exercise lookup hits, lookup misses, `use bio as b;` makes `b.`
  resolve and `bio.` error with a "did you mean `b`?" hint, the
  namespace-identifier reservation rule (`func bio() {}` errors after
  bare `use bio;`, passes after aliased `use bio as b;`), `use`-gating
  before any qualified call is allowed, and dot-separated constants.
- **Formatter:** round-trip a program containing qualified calls;
  verify no space around `.`.
- **AST JSON dump:** emit `QualifiedCallExpr` for a synthetic source.
- **REPL:** `EvalInteractive` path through a qualified call; multi-line
  continuation crossing a `.` boundary.
- **End-to-end:** one test in `cmd/jennifer/examples_test.go` (or a
  new `namespace_test.go`) registers a test-only namespaced library
  and runs a small source string; no `examples/*.j` golden file
  required.
- **Regression:** the five shipping essentials act as the "namespaces
  don't break flat builtins" sanity suite - their existing tests
  must keep passing unchanged.

**Docs to write at end of M8:**

- Update `docs/libraries/index.md` with the namespace policy and a
  rule for library authors ("is your library essential or domain?
  if domain, set a namespace").
- Add `docs/libraries/os.md` for the minimal `os` slice shipping
  in M8 (covers `os.platform()`, `os.getEnv(name)`, `os.JENNIFER_LF`,
  `os.JENNIFER_OS`); expanded again in M13.1.
- Update `docs/technical/grammar.md` EBNF.
- Update `docs/technical/interpreter.md` Builtins-and-libraries
  section to describe the lookup rule.
- New section in `docs/user-guide/style-guide.md`: how to write namespaced calls
  (`bio.translate($seq)`, no space around `.`, same as method-call
  paren-hugging).
- `cmd/jennifer/fmt.go` already preserves `TOKEN_DOT` verbatim;
  verify the formatter handles it cleanly with a small test.

**Exit criterion:** the shipping `os` slice can be `use`d (with and
without aliasing) and exercises both qualified calls and qualified
constants; synthetic namespaced builtins prove the mechanism
generalises; the parser, interpreter, REPL, AST JSON dump, and
formatter all handle the qualified form correctly; the five
existing flat essential libraries continue to work unchanged.

---

## M9 - Collection operations

The `list`/`map` types shipped in M6 with literals, indexing, and
iteration but no manipulation helpers. M9 adds two opt-in libraries
that cover the rest:

- **`lists` library** (`use lists;`): `push`, `pop`, `first`, `last`,
  `head`, `tail`, `reverse`, `sort`, `contains`, `concat`, `slice`.
  All functions return a **new list** (no mutation through reference;
  matches Jennifer's value semantics) - callers write
  `$xs = push($xs, item);`. `sort` works on primitive element types;
  comparator-based sort is deferred until methods are first-class
  values.
- **`maps` library** (`use maps;`): `keys`, `values`, `delete`,
  `merge`. Same pattern - functions return a new map, no in-place
  mutation. `delete($m, key)` returns a shrunk map; `merge($a, $b)`
  returns a new map with `$b` overlaying `$a`.

**Sugar: `$xs[] = item;` syntax-level append.** For the 80% case of
"build a list by appending", introduce a new write target where `[]`
means "the position just past the end". Mechanically the same as
`push($xs, item)` but cheaper to write. Reuses index-assign machinery
(the binding gets rewritten to the extended list); no new keyword. The
parser learns to accept `LBRACKET RBRACKET` as a special index in
write position. Reads of `$xs[]` are not valid (no obvious meaning).

**`core.has` stays map-only**, as already documented. List
membership is `lists.contains($xs, item)` - matches the
`strings.contains($s, $sub)` shape (haystack first, needle second; PHP's
`in_array($needle, $haystack)` argument order is famously confusing
and deliberately not adopted).

**Naming convention** (resolved at start of M9): flat names within
each library following the existing `strings.contains` /
`strings.startsWith` style. No `array_*` PHP-style prefix. Once M8
namespacing lands, qualified `lists.contains($xs, x)` form becomes
available; until then, `use lists;` + bare `contains` works.

**Why now rather than ship with M6**: M6 already pushed a lot of new
machinery (parser, runtime, type stamping, formatter rules). Punting
helpers gave us time to confirm the value-semantics + return-new-list
shape against real example code (the showcase, wordcount). With that
validated, M9 is mostly stdlib registration: each function is one Go
helper. The `$xs[] = item;` sugar is the only language-level change.

**Implementation notes:**

- Each library function is a builtin: `Register("lists", "push", pushFn)`.
- `pushFn` takes `(list, item) -> list of T` - copies the input list,
  appends the item, returns it. The caller's assignment writes the
  result back; the original input is untouched. Cost: O(n) per push,
  acceptable at Jennifer's scale; copy-on-write is a future optimization.
- The `$xs[] = item;` sugar parses as a new AST node `AppendStmt {
  Target *VarExpr, Value Expr }`. Interpreter walks identically to
  IndexAssignStmt but writes one slot past the end.
- `sort` uses Go's `sort.Slice` with a kind-aware less function; mixed
  types in one list error (kind already enforced at element-write
  time, so this should never happen in practice).

**Docs to update:**

- `docs/libraries/lists.md` (new file) - covers every function with
  signature, semantics, example, and the "returns new list" rule.
- `docs/libraries/maps.md` (new file) - same pattern.
- `docs/libraries/index.md` - add the two new entries to the catalog.
- `docs/user-guide/imports.md` - mention `lists` / `maps` in the libraries
  table; cover the `$xs[] = ...;` sugar in the Lists and maps section.
- `docs/technical/grammar.md` - new EBNF for the append form and
  new `AppendStmt` AST node.
- `docs/user-guide/style-guide.md` - no whitespace inside `$xs[]` (consistent
  with `$xs[0]`).
- `examples/showcase.j` and/or `examples/wordcount.j` - exercise
  push/pop/sort/contains so the goldens cover the new surface.

**Exit criterion:** the showcase exercises `push`, `pop`, `sort`,
`contains`, and the `$xs[]` sugar end-to-end; round-trips through
`fmt` are stable; `lists` and `maps` libraries both show up in
`jennifer help`'s "available libraries" error list.

---

The next phase splits into three arcs: Phase A finishes the
language so libraries have something to stand on, Phase B ships
the foundational libraries that every Jennifer program needs,
Phase C ships I/O libraries, and Phase D ships the higher-level
ecosystem (Jennifer-coded libraries, the module system that
unblocks them, crypto, a server). Phase E (WASM and specialised
domains) is the long horizon.

The library milestones use sub-numbering (M13.1, M13.2, ...) so
each library ships and is reviewed independently. This is the
first time we use sub-milestones; the practice is justified
because each library is small enough to land in a single sitting
once the language foundation is in place.

---

**Phase A: language completion (M10-M12).** These three milestones
close the biggest daily-use gaps and add the foundational types every
later library needs.

## M10 - Control-flow completion

- `break;` and `continue;` inside `while`/`for`/`for-each`/`repeat`.
- `repeat { } until (cond);` post-test loop. New keywords `repeat`
  and `until`. `until` makes condition inversion explicit, matching
  the word-operator style (`and`/`or`/`not`); `do { } while ...`
  considered and rejected.
- `exit;` and `exit EXPR;`. Process exit; integer expression sets
  the OS exit code on top-level execution. Distinct from `return`
  (which stays method-scoped).
- Bundled: printf `%s|align=center` while the modifier table is
  being touched anyway.

## M11 - Bytes and bit operators

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
- New `bytes <-> string` codecs (UTF-8 by default; lossless
  re-encoding lives in the future `encoding` library).
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

## M12 - Structs / records

- New `def struct Name { field as type, ... };` syntax (working
  name; revisited at start of M12).
- Literals: `Name{ field: expr, ... }`. Field access: `$p.field`.
  Value semantics like lists/maps - `def $q as Name init $p;`
  copies. `const` is deep.
- Unblocks every library that wants to return composite data
  (file info, time values, network endpoints, http request /
  response).

---

**Phase B: foundational libraries (M13.x).** Small, frequently-used
libraries grouped under M13 with sub-numbering; each sub-milestone
ships one library independently.

## M13.1 - `os`

Expands the M8 demo slice (`os.platform()`, `os.getEnv(name)`,
`os.JENNIFER_LF`, `os.JENNIFER_OS` already ship there). Adds:
process args (`os.args -> list of string`), exit code
(`os.exit(n)`), and the rest of the `JENNIFER_*` constants
(`os.JENNIFER_BUILD`, `os.JENNIFER_PLATFORM`,
`os.JENNIFER_ARCHITECTURE`).

## M13.2 - `random`

Non-crypto pseudo-random. `random.rand()` returns float in
`[0, 1)`. `random.randInt(lo, hi)`. `random.seed(n)` for
reproducibility. Crypto-grade random ships separately in M17.

## M13.3 - `time`

New `time` value type. `time.now()`, formatting, parsing,
arithmetic on durations. Larger than other M13 entries because
the type is new; may split into M13.3.x if scope demands.

## M13.4 - `hash` and `crc`

Common digests over `bytes` (MD5, SHA-1, SHA-256, CRC32,
CRC64). Pure compute, no dependencies. Crypto-relevant primitives
that don't require key material live here; key-based crypto goes
in M17.

## M13.5 - `encoding`

`encoding.isAscii`, `encoding.lenBytes`, `encoding.lenRunes`,
`encoding.toBytes($s, $codec)`, `encoding.toString($bytes,
$codec)`, hex / base64 helpers. Operates at the `bytes <->
string` boundary.

---

**Phase C: I/O libraries (M14.x).** System libraries that touch the
OS or do significant compute.

## M14.1 - `fs`

File I/O. `fs.read`, `fs.write` for `string` and `bytes`,
`fs.stat` returning a struct, directory walk. Requires M11
(bytes) and M12 (structs). Brings the M7-deferred file handles
and any non-stdin input source (`fs.open(path) -> handle`,
`handle.readLine()`, etc.) under one library.

## M14.2 - `net`

Sockets. Plain TCP and UDP first; TLS deferred to its own
sub-milestone. The first place a real `bytes` round-trip
matters.

## M14.3 - `regex`

Regular expressions over `string`. Large standalone milestone;
pure string processing, no other dependencies.

---

**Phase D: higher-level and Jennifer-coded libraries (M15-M18).**

## M15 - Module system for Jennifer-coded libraries

Real modules so `.j` libraries get namespaces, scope, and
explicit exports. Lands the form M8 deferred
(`import "templateengine.j" as ninja;` with a real module
boundary; today `import "x.j";` is a textual splice with no
namespace). File imports stay as textual splice for inline
composition (same source file); library distribution moves to
modules.

## M16.x - Jennifer-coded essentials

Built atop the existing system libraries. Sub-milestones in
priority order:

- **M16.1 - `http`** (client) - atop `net`.
- **M16.2 - `json`** - data interchange ubiquity.
- **M16.3 - `csv`** - simple, useful early.
- **M16.4 - `yaml`, `xml`, `markdown`, `pretty`** - one or more
  sub-milestones depending on scope when planned.

## M17 - `crypto`

Symmetric and asymmetric primitives, key derivation,
crypto-grade random. System library; TinyGo-safe primitives
only. Hashes already shipped in M13.4.

## M18 - `httpd`

Pure Jennifer HTTP server atop `net`. The point where Jennifer
becomes useful for serving content.

---

**Phase E: WASM and specialised domains (M19+).** Not committed to a
timeline; recorded so the design doesn't foreclose them.

## M19 - WASM runtime embedding

Wazero or similar inside the interpreter binary. TinyGo-size
cost evaluated honestly before commitment. Without M19, no WASM
libraries.

## M20.x - WASM libraries

If M19 ships, sandboxed plugins via `use wasm:libname;`. Each
library a sub-milestone.

## M21+ - Specialised domains

Each domain its own milestone with sub-milestones as needed:

- **ML.** Vector, matrix, stats, ML primitives.
- **Bioinformatics.** Sequence alignment (Smith–Waterman,
  Needleman–Wunsch), FASTA/FASTQ parsers, molecule structures.
- **Sandbox.** Restricted-capability execution.

Ordered when demand surfaces. WASM libraries (M20.x) may cover
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
- **`fmt` comment preservation.** Today the lexer drops `#` and
  `/* */` comments; the formatter would need to carry them as
  tokens (or attach them to following nodes) and weave them back
  in. Deferred from M5.
- **`fmt` blank-line preservation or auto-insertion.** Either keep
  user blank lines as a side channel during lexing, or insert them
  automatically between logical groups (imports vs. methods vs.
  top-level code, method-to-method). Deferred from M5.
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
