# `json` - JSON encode / decode

Enable with `use json;`. RFC 8259 JSON, mapped onto Jennifer's value
model. Hand-rolled (no host `encoding/json`) so it works identically on
both binaries.

## Surface

| Call                     | Returns  | Notes                                                          |
| ------------------------ | -------- | -------------------------------------------------------------- |
| `json.encode(v)`         | `string` | Compact JSON for any encodable value.                          |
| `json.encodePretty(v)`   | `string` | Same, 2-space indented; empty arrays / objects stay `[]`/`{}`. |
| `json.decode(s)`         | *value*  | Parse JSON text into a generic value (see below).              |

## Encoding

`json.encode` walks the value and writes its JSON image:

| Jennifer                | JSON                                             |
| ----------------------- | ------------------------------------------------ |
| `null`                  | `null`                                           |
| `bool`                  | `true` / `false`                                 |
| `int`                   | integer number                                   |
| `float`                 | number (always with a `.` or exponent, so it decodes back as `float`) |
| `string`                | string (escaped)                                 |
| `bytes`                 | base64 string                                    |
| `list of T`             | array                                            |
| `map of string to V`    | object (in insertion order)                      |
| struct                  | object, keys in field-declaration order          |

Encode errors (positioned at the call): a `map` with a non-string key, a
`task` value, or a non-finite `float` (`NaN` / infinity have no JSON
image).

```jennifer
use io;
use json;

io.printf("%s\n", json.encode({"id": 1, "tags": ["a", "b"], "ok": true}));
# {"id":1,"tags":["a","b"],"ok":true}
```

## Decoding

`json.decode` produces **generic values**: a JSON object becomes a
`map of string to V`, an array a `list of V`, a number an `int` when it
has no fractional or exponent part (else a `float`), and `true`/`false`/
`null`/strings map directly. Malformed input is a positioned runtime
error carrying the line and column within the JSON text (catchable with
`try`/`catch`).

```jennifer
def m as map of string to int init json.decode("{\"x\": 7, \"y\": 8}");
io.printf("%d\n", $m["x"]);   # 7
```

### Heterogeneous objects

A decoded collection is validated entry by entry against the declared
element type at the binding - the same check a literal gets. So a
*homogeneous* JSON object binds to the matching typed map (above), but a
*heterogeneous* one (mixed value types) matches no single
`map of string to V` and is **rejected**, rather than silently holding a
value of the wrong type:

```jennifer
# error: 8 is an int, not a string
def m as map of string to string init json.decode("{\"x\": \"7\", \"y\": 8}");
```

Until a general `any` type lands, hold heterogeneous data by rebuilding
the parts you need into a struct (below), or decode only homogeneous
pieces.

### No map-to-struct coercion

Jennifer does no coercion at binding boundaries (you write `5.0`, not
`5`, for a float), and JSON decode is no exception: `json.decode` returns
a map, never a struct. To land JSON in a typed struct, rebuild it
explicitly - the schema is right there in the literal:

```jennifer
def struct Point { x as int, y as int };
def m as map of string to int init json.decode("{\"x\": 7, \"y\": 8}");
def p as Point init Point{ x: $m["x"], y: $m["y"] };
```

The encode direction has no such restriction - `json.encode($p)` serializes
a struct to an object directly. See
[technical/rejected.md](../technical/rejected.md) for why the decode
coercion was declined.

## See also

- [encoding.md](encoding.md) - hex / base64 and character-set codecs
  (`json` reuses base64 for `bytes`).
- [cheatsheet.md](cheatsheet.md) - every builtin at a glance.
