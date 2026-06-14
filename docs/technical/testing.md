# Testing

| Package                | What it tests                                                                                                                                                                                                          |
| ---------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `internal/lexer`       | Token-by-token output for fixed inputs; error cases                                                                                                                                                                    |
| `internal/parser`      | AST shape via `Sprint`; precedence; error cases                                                                                                                                                                        |
| `internal/interpreter` | Full programs in-memory; stdout captured                                                                                                                                                                               |
| `internal/lib/io`      | `Install` registers `printf`/`sprintf`; arity and format errors                                                                                                                                                        |
| `internal/lib/meta`    | `meta.VERSION` matches `version.Version`; `meta.BUILD` matches the compiler tag                                                                                                                                        |
| `internal/lib/time`    | Constructors / accessors / arithmetic round-trip; ISO weekday remapping; deterministic via a `nowFunc` package-var override                                                                                            |
| `cmd/jennifer`         | Golden test that runs every `examples/*.j` and compares stdout to `examples/expected/*.txt`; REPL `inputComplete` helper; AST-JSON validity; formatter idempotence + behavior preservation; cross-file error reporting |

Run everything with `go test ./...`.
