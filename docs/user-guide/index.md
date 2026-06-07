# Jennifer - User Guide

Jennifer is a small, experimental, interpreted programming language. This
guide covers everything you can do in Jennifer today
([Milestone 6](../milestones.md)).

## Contents

- [Installing & running](installing.md) - building the binary, running a
  program, the interactive REPL, the inspection and formatting commands.
- [Your first program](first-program.md) - a one-screen `hello.j` and a
  walkthrough of what each line does.
- [Syntax](syntax.md) - tokens, comments, identifier rules.
- [Types and values](types-and-values.md) - the primitive types,
  variables and constants, scoping rules, and the compound types
  `list` and `map`.
- [Methods](methods.md) - declaring methods, parameters, return values,
  recursion, hoisting, the no-shadowing rule for builtins.
- [Control flow](control-flow.md) - operators, precedence, `if` / `while`
  / `for` / for-each.
- [Imports](imports.md) - `use LIB;` for libraries, `import "file.j";` for
  source files, the library catalog.
- [Examples](examples.md) - small programs that exercise the language
  end-to-end.
- [Style guide](style-guide.md) - the canonical source style that
  `jennifer fmt` produces; the spacing/indentation rules, the names
  convention, the `[]` and `{}` literal layout.

## Related

- [docs/libraries/](../libraries/index.md) - one reference page per
  standard library (`io`, `convert`, `math`, `strings`, `core`).
- [docs/technical/](../technical/index.md) - interpreter internals for
  contributors.
- [docs/milestones.md](../milestones.md) - what's shipped, what's
  planned.
