# Rejected features

Proposals that were considered and explicitly turned down. Recorded here so
the same ideas don't come back as fresh suggestions next session.

## Increment / decrement (`++`/`--`)

Considered: postfix `$i++` and prefix `++$i`.

Rejected because:

- The pre/post distinction is a real footgun - the two forms differ only
  in expression context, which is exactly where bugs hide. Swift removed
  `++`/`--` in version 3 (2016) for this exact reason.
- The savings are tiny (three characters) and only apply to `+1` / `-1`.
- Python rejected them from the start and the language hasn't suffered.
- `$i = $i + 1;` is verbose but unambiguous; the readability cost is small.

## Compound assignment (`+=`, `-=`, `*=`, `/=`, `div=`, `%=`)

Considered as an alternative to `++`/`--`.

Rejected because:

- `div=` reads particularly badly - mashing a word-operator into the
  assignment-operator family stands out.
- Several operators to add and remember for marginal ergonomic gain over
  `$x = $x + E;`.
- Slippery slope: would we also need a string-concat `+=`? An `and=`?
  Where does the family end?
- Keeping a single assignment shape (`$x = EXPR;`) makes source code uniform
  and matches Jennifer's "one way to do each thing" stance.
