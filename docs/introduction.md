# Jennifer

Jennifer is a small, experimental, interpreted programming language
written in (Tiny)Go and ships as two binaries:

- **`jennifer`** - standard Go build, full host-feature surface.
  This is the default binary you install and reach for.
- **`jennifer-tiny`** - TinyGo build, smaller and embeddable.
  Missing `os/exec` and the network stack (TinyGo runtime gaps);
  calls into those surfaces return a friendly runtime error
  pointing back at `jennifer`. Use this variant when binary size
  or embeddability matters (embedded systems, minimal containers,
  small-footprint scripting hosts).

But small is not bare. Jennifer is batteries-included: a broad standard
library and a growing set of distributable modules cover what real programs
actually need, so you build genuine tools, not toys. Text handling has full
[regular expressions](libraries/regex.md); structured data flows through
[JSON](libraries/json.md); email is a complete stack -
[SMTP](modules/smtp.md) to send, [POP3](modules/pop.md) and
[IMAP](modules/imap.md) to receive; in-memory data stores come through
[Redis](modules/redis.md) and [memcached](modules/memcache.md) clients; the
web runs from an ergonomic [REST client](modules/rest.md) to turnkey
integrations such as [Gotify](modules/gotify.md) push notifications; and
lightweight [concurrency](user-guide/concurrency.md) is built into the language
via `spawn` and the [task](libraries/task.md) library. Browse the full
[library catalog](libraries/index.md) and
[module catalog](modules/index.md) - both grow with every release.

It is also a natural fit for teaching and learning: an interactive
[REPL](technical/cli_repl.md), an [easy-to-read grammar](technical/grammar.md),
and [token and AST dumps](technical/cli_inspect.md) that make it ideal for
mastering language design, plus a built-in [linter](technical/cli_lint.md) and
[profiler](technical/cli_profile.md) and full
[test support](technical/cli_test.md). Its
[strict, explicit design](design-stances.md) - conditions must be `bool`,
conversions are spelled out, names never shadow, and errors are positioned -
surfaces a mistake as a clear message instead of a silent surprise, so a
learner sees exactly what went wrong.

Source files use the `.j` extension. Whitespace is not significant
anywhere; statements end with `;`.

```jennifer
use io;
use time;

def start as time.Time init time.now();
io.printf("hello, world\n");
def gap as time.Duration init time.sub(time.now(), $start);
io.printf("ran for %d ms\n", time.milliseconds($gap));
```

## Write Jennifer with your editor and an AI assistant

Syntax highlighting ships in
[editors/](https://github.com/mplx/jennifer-lang/tree/main/editors) (a
Vim / Neovim drop-in, a TextMate grammar for VS Code / Sublime / Zed, a
highlight.js definition). And because Jennifer is new, we ship
[`JENNIFER.md`](https://github.com/mplx/jennifer-lang/blob/main/JENNIFER.md):
drop it into your project and point an AI coding assistant at it ("we
code in Jennifer, see JENNIFER.md, let's go") so it writes correct `.j`
from the start. See [Editor & AI support](user-guide/tooling.md).

## What's in this site

- **[Getting started](user-guide/installing.md)** - install the
  interpreter and run your first program.
- **[Editor & AI support](user-guide/tooling.md)** - highlighting and
  the drop-in `JENNIFER.md` for AI-assisted coding.
- **[Language reference](user-guide/index.md)** - syntax, types,
  methods, control flow, imports, style.
- **[Libraries](libraries/index.md)** - per-library reference plus
  an alphabetical [cheatsheet](libraries/cheatsheet.md) of every
  builtin.
- **[Technical reference](technical/index.md)** - implementation
  details for the lexer, parser, interpreter, and CLI.
- **[Project](milestones.md)** - milestones, design stances,
  glossary.

## Status

Pre-1.0. The major version stays at `0.x.y` while the language is
still finding its shape; breaking changes can happen at milestone
boundaries and are called out in
[docs/milestones.md](milestones.md). Once Jennifer reaches `1.0.0`,
semver applies and breaking changes need a major bump.

## Source

Source, issues, and pull requests live at
<https://github.com/mplx/jennifer-lang>.

License: LGPL-3.0-only.
