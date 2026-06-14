# `encoding` - byte/string introspection + character codecs

Enable with `use encoding;`. Three things in one library:

1. **Introspection helpers** for telling rune count from byte count and
   asking whether a byte sequence is ASCII.
2. **Binary-to-text encodings** (`toText` / `fromText`) for hex and
   base64 (standard + URL-safe).
3. **Character codecs** (`encode` / `decode`) for converting Jennifer
   strings into single-byte legacy encodings and back.

The cross-kind UTF-8 codec ships with [`convert`](convert.md)
(`convert.bytesFromString` / `convert.stringFromBytes`); this library
is where the codec proliferation belongs because that's where the
table-based implementations live.

```jennifer
use io;
use convert;
use encoding;

def src as bytes init convert.bytesFromString("café", "utf-8");
io.printf("hex = %s\n", encoding.toText($src, "hex"));            # 636166c3a9
io.printf("ascii? %t\n", encoding.isAscii($src));                 # false
io.printf("latin1 = %x\n", encoding.encode("café", "latin-1"));   # 63 61 66 e9
```

## Introspection

| Call                       | Returns | Notes                                                                            |
| -------------------------- | ------- | -------------------------------------------------------------------------------- |
| `encoding.isAscii(b)`      | `bool`  | True iff every byte < 0x80. Empty bytes is true.                                 |
| `encoding.lenBytes(s)`     | `int`   | Byte length of `s`'s UTF-8 encoding. Pair with `len(s)` (rune count).            |
| `encoding.lenRunes(b)`     | `int`   | Decoded rune count of valid UTF-8 `bytes`; errors on invalid UTF-8.              |

## Binary-to-text: `toText` / `fromText`

Hex and base64 share one verb pair with the format as a string. The
format string is also the place url-safe base64 lives - no separate
`base64Url` verb to remember.

| Call                       | Returns  | Notes                                                       |
| -------------------------- | -------- | ----------------------------------------------------------- |
| `encoding.toText(b, fmt)`  | `string` | `fmt`: `"hex"`, `"base64"`, `"base64-url"`.                 |
| `encoding.fromText(s, fmt)`| `bytes`  | Same `fmt` set. `"hex"` accepts both upper and lower case.  |

The format strings are case-sensitive (unlike codec names, which
normalise). Unknown formats error with the supported set listed:
`encoding.toText: unknown text format "morse"; known: "hex", "base64", "base64-url"`.

**Why a format-string table instead of `encoding.hex()` / `encoding.base64()`?**
Jennifer's letters-only identifier rule rejects digits in method names
(`encoding.base64` won't parse), and the codec-table shape is already
how the rest of this library and [`convert`](convert.md), [`hash`](hash.md),
[`crc`](crc.md) work.

## Character codecs: `encode` / `decode`

Pair of functions parameterised by a codec string. `encode` turns a
Jennifer string into the named encoding's byte representation;
`decode` is the inverse.

| Call                         | Returns  | Notes                                                                |
| ---------------------------- | -------- | -------------------------------------------------------------------- |
| `encoding.encode(s, codec)`  | `bytes`  | Errors when a rune doesn't fit in the codec (e.g. `€` in ASCII).     |
| `encoding.decode(b, codec)`  | `string` | Errors when a byte has no mapping in the codec.                      |
| `encoding.codecs()`          | `list of string` | The canonical codec names in registration order.             |

### Codec set shipped in M15.7

| Codec string     | Notes                                                                                              |
| ---------------- | -------------------------------------------------------------------------------------------------- |
| `"ascii"`        | 7-bit ASCII. Rejects any byte >= 0x80 on decode and any rune >= 0x80 on encode.                    |
| `"latin-1"`      | Identity map for U+0000..U+00FF. Same codec as `"iso-8859-1"`.                                     |
| `"windows-1252"` | Latin-1 except positions 0x80..0x9F carry the printable "smart quotes" set (incl. EURO SIGN). Five positions (0x81, 0x8D, 0x8F, 0x90, 0x9D) are canonically undefined and error on encode / decode. |
| `"ebcdic"`       | IBM Code Page 1047 (Open Systems Latin-1 EBCDIC variant).                                          |

The long-tail single-byte codecs (`iso-8859-{2..16}`,
`windows-{1250,1251,1253..1258}`) are parked in **M24+** to be
picked up when a real program asks for one - they're just rows
in the same table when added.

### Codec name normalisation

Codec names are case-insensitive and ignore `-`, `_`, and spaces, so
all of these resolve to the same codec:

| Input         | Resolves to    |
| ------------- | -------------- |
| `"ASCII"`     | `"ascii"`      |
| `"us-ascii"`  | `"ascii"`      |
| `"latin-1"`   | `"latin-1"`    |
| `"latin_1"`   | `"latin-1"`    |
| `"ISO-8859-1"`| `"latin-1"`    |
| `"iso88591"`  | `"latin-1"`    |
| `"cp1252"`    | `"windows-1252"` |
| `"MS 1252"`   | `"windows-1252"` |
| `"IBM-1047"`  | `"ebcdic"`     |
| `"cp1047"`    | `"ebcdic"`     |

`encoding.codecs()` returns the canonical form (hyphenated lowercase).

## Errors

- Wrong arity: `encoding.encode expects 2 arguments (string, codec), got 1`.
- Wrong scalar type:
  `encoding.encode: first argument must be string, got bytes`.
- Unknown codec:
  `encoding.encode: unknown codec "klingon"; known: "ascii", "latin-1", "windows-1252", "ebcdic"`.
- Unrepresentable rune:
  `encoding.encode (ascii): rune U+00E9 at byte position 3 is outside ASCII (0x00..0x7F)`.
- Byte with no codec mapping (Windows-1252 undefined positions):
  `encoding.decode (windows-1252): byte 0x81 at position 0 has no mapping in windows-1252`.
- Invalid UTF-8 to `lenRunes`:
  `encoding.lenRunes: input is not valid UTF-8`.

All errors are positioned at the call site.

## See also

- [convert.md](convert.md) - the UTF-8 pair (`bytesFromString` /
  `stringFromBytes`).
- [hash.md](hash.md), [crc.md](crc.md) - digest libraries whose
  output bytes you'll often `encoding.toText($digest, "hex")`.
- [milestones.md](../milestones.md) - M15.7 surface and the long-tail
  codecs parked in M24+.
