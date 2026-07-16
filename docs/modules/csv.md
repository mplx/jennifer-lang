# `csv` - RFC 4180 comma-separated values

Import with `import "csv.j" as csv;`. Parses CSV text into rows of string
fields and formats rows back into text, with a quoting-aware hand-written
scanner. Pure Jennifer over `strings` and `maps`, so it runs on either
binary. The delimiter is configurable, so the same code reads and writes
TSV and other single-character-separated formats.

```jennifer
use io;
import "csv.j" as csv;

def rows as list of list of string init csv.parse("name,note\n\"Smith, J\",hi");
io.printf("%s | %s\n", $rows[1][0], $rows[1][1]);        # Smith, J | hi

def recs as list of map of string to string init csv.toRecords($rows);
io.printf("%s\n", $recs[0]["note"]);                      # hi
```

Runnable: [`examples/modules/csv_demo.j`](https://github.com/jennifer-language/jennifer/blob/main/examples/modules/csv_demo.j).

## Surface

| Call                              | Returns                          | Notes                                                                        |
| --------------------------------- | -------------------------------- | ---------------------------------------------------------------------------- |
| `csv.parse(s)`                    | `list of list of string`         | Parse standard comma-delimited CSV into rows of fields.                      |
| `csv.parseWith(s, delim)`         | `list of list of string`         | Same, with a single-character delimiter (`"\t"` for TSV).                    |
| `csv.format(rows)`                | `string`                         | Encode rows as comma-delimited CSV; quotes fields that need it.             |
| `csv.formatWith(rows, delim)`     | `string`                         | Same, with a single-character delimiter.                                     |
| `csv.toRecords(rows)`             | `list of map of string to string`| Treat row 0 as a header; map each later row to a header-keyed record.        |
| `csv.fromRecords(header, records)`| `list of list of string`         | Inverse: a header row followed by one row per record, in `header` order.     |

## Parsing (RFC 4180)

`parse` and `parseWith` implement the [RFC 4180](https://www.rfc-editor.org/rfc/rfc4180)
rules:

- Fields are separated by the delimiter (a comma by default); records by
  `LF` or `CRLF`. A bare `CR` outside quotes also ends a record.
- A field wrapped in `"` may contain the delimiter, line breaks, and
  quotes; an embedded quote is written doubled (`""`) and decodes to one.
- An empty input yields no rows; a trailing record separator does **not**
  add an empty trailing row. A separator with nothing after it *within* a
  record is a real empty field (`a,` is two fields, the second empty).

```jennifer
# Embedded comma, doubled quote, and newline all survive.
def rows as list of list of string init csv.parse("\"Smith, J\",\"said \"\"hi\"\"\",\"two\nlines\"");
# rows[0] == ["Smith, J", "said \"hi\"", "two\nlines"]
```

## Formatting

`format` / `formatWith` are the inverse. A field is quoted only when it
carries the delimiter, a quote, or a line break; embedded quotes double.
Records are joined with `LF` and **no trailing newline**, so
`parse(format(rows))` round-trips the data:

```jennifer
def rows as list of list of string init [];
$rows[] = ["plain", "has,comma", "q\"uote"];
io.printf("%s\n", csv.format($rows));
# plain,"has,comma","q""uote"
```

Only the **record separators** normalise: a `CRLF`- or `CR`-terminated
input re-emits with `LF` between records. Line breaks *inside* a quoted
field are field content and pass through verbatim, so no data is altered.

## Header-keyed records

Most CSV has a header row. `toRecords` pairs it with the data rows, giving
one `map of string to string` per record keyed by column name; `fromRecords`
rebuilds rows from records and an explicit header:

```jennifer
def rows as list of list of string init csv.parse("name,age\nAda,36\nGrace,45");
def recs as list of map of string to string init csv.toRecords($rows);
# recs[0] == {"name": "Ada", "age": "36"}

def back as list of list of string init csv.fromRecords(["name", "age"], $recs);
# back == [["name","age"], ["Ada","36"], ["Grace","45"]]
```

Details worth knowing:

- Every record carries **every** header key. A data row shorter than the
  header fills the missing fields with `""`; fields past the header width
  are dropped (they have no name).
- Duplicate header names collapse - a later column overwrites an earlier
  one of the same name (map keys are unique).
- `fromRecords` takes the header **explicitly** rather than reading it off
  the records, because map iteration order is insertion order per record
  and would not give a stable column order across records. A key absent
  from a record writes `""`.
- `toRecords([])` is `[]`; a header-only input yields an empty record list.

## Out of scope

Type inference (numbers, booleans, dates) is not part of this module -
every field is a `string`, and the caller converts what it needs with
`convert.toInt` / `convert.toFloat`. Streaming a file too large to hold in
memory is also out of scope: `parse` takes a whole string. Read the file
with `fs.readString` (or slurp stdin) and hand the text in.

## See also

- [strings.md](../libraries/strings.md) - `split` / `join` / `replace`,
  which `csv` builds the scanner and encoder on.
- [maps.md](../libraries/maps.md) - `has` / `keys`, used by the record
  helpers.
- [fs.md](../libraries/fs.md) - `readString` to load a CSV file to hand to
  `parse`.
- [modules/index.md](index.md) - the module catalog and import rules.
