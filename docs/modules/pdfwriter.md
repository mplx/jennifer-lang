# `pdfwriter` - generate simple PDF documents

Import with `import "pdfwriter.j" as pdf;`. Build a `Document` of `Page`s with
value-semantic builders - text, lines, rectangles - then `render()` writes the
PDF object / xref structure by hand (no stdlib PDF) as `bytes`, the way
[`htmlwriter`](htmlwriter.md) / [`label`](label.md) generate their formats.
Content streams are FlateDecode-compressed via [`compress`](../libraries/compress.md).
Pure Jennifer; runs on **both** binaries.

```jennifer
import "pdfwriter.j" as pdf;
use fs;

def p as pdf.Page init pdf.page(612, 792);
$p = pdf.text($p, 72, 720, "Helvetica", 24, "Hello, PDF");
def doc as pdf.Document init pdf.addPage(pdf.document(), $p);
fs.writeBytes("out.pdf", pdf.render($doc));
```

Runnable: [`examples/modules/pdfwriter_demo.j`](https://github.com/jennifer-language/jennifer/blob/main/examples/modules/pdfwriter_demo.j).

## Coordinates and units

Coordinates are in **PDF points** (1/72 inch), with the origin at the
**bottom-left** and y increasing **upward**. All coordinates and sizes are
integers. Common page sizes: US Letter `612 x 792`, A4 `595 x 842`. Colours are
`0-255` RGB integers.

## Building

Every builder is **value-semantic** - it returns a fresh copy and never mutates
its argument, so you thread them (`$p = pdf.text($p, ...)`).

| Call | Returns | |
| ---- | ------- | - |
| `pdf.document()` | `Document` | an empty document |
| `pdf.page(width, height)` | `Page` | a blank page of the given size |
| `pdf.text(pg, x, y, font, size, str)` | `Page` | draw text at (x, y) |
| `pdf.line(pg, fromX, fromY, toX, toY)` | `Page` | draw a stroked line |
| `pdf.rect(pg, x, y, width, height, filled)` | `Page` | draw a rectangle (fill or stroke) |
| `pdf.color(pg, red, green, blue)` | `Page` | set fill + stroke colour for what follows |
| `pdf.addPage(doc, pg)` | `Document` | append a page |
| `pdf.render(doc)` | `bytes` | the finished PDF |

`color` sets the drawing colour for **subsequent** operations on that page (both
fill and stroke), so order matters: set the colour, then draw. `rect`'s `filled`
flag fills the rectangle when `true`, otherwise strokes its outline.

## Fonts

`text` takes one of the **standard-14** base fonts every PDF viewer provides;
any other name throws `Error{kind: "pdfwriter"}`:

```
Helvetica  Helvetica-Bold  Helvetica-Oblique  Helvetica-BoldOblique
Times-Roman  Times-Bold  Times-Italic  Times-BoldItalic
Courier  Courier-Bold  Courier-Oblique  Courier-BoldOblique
Symbol  ZapfDingbats
```

Each distinct font used becomes one shared Type1 font object
(`WinAnsiEncoding`). Text is escaped for the PDF literal-string syntax (`\`, `(`,
`)`, and line breaks), so any ASCII / Latin-1 string is safe to pass.

## Metadata

Set document metadata - the PDF Info dictionary shown in a viewer's "Document
Properties" - with `pdf.info(doc, key, value)`. `key` is a PDF Info key:

| Key | |
| --- | - |
| `Title` / `Author` / `Subject` / `Keywords` | the descriptive fields |
| `Creator` | the app that authored the source |
| `Producer` | the app that wrote the PDF (defaults to `"Jennifer pdfwriter"`) |
| `CreationDate` / `ModDate` | PDF date strings (see `pdfDate` below) |

```jennifer
def doc as pdf.Document init pdf.document();
$doc = pdf.info($doc, "Title", "Q3 Report");
$doc = pdf.info($doc, "Author", "Ada Lovelace");
$doc = pdf.info($doc, "Keywords", "report, finance, q3");
```

`document()` presets `Producer` to `"Jennifer pdfwriter"`; every other field is
unset until you set it. Any custom key works too. Dates use the PDF date syntax,
which `pdf.pdfDate(t)` builds from a `time.Time`:

```jennifer
use time;
$doc = pdf.info($doc, "CreationDate", pdf.pdfDate(time.utc()));   # D:20260714160000+00'00'
```

## Rendering

`render(doc)` produces a complete PDF 1.7 file as `bytes`: a catalog, a page
tree, one page dict + one FlateDecode-compressed content stream per page, the
shared font objects, an Info dictionary when any metadata is set, a
cross-reference table with correct byte offsets, and the trailer. Write it with
`fs.writeBytes`, return it from an `httpd` handler, or attach it via `mime`. It
validates clean under `qpdf --check`.

**Byte-identical output.** The same document always renders to the **exact same
bytes** - on either binary, run to run. This is deliberate: `pdfwriter` never
auto-stamps a `CreationDate` or any other timestamp (you opt into one explicitly
via `info` + `pdfDate`), so nothing varies with wall-clock time. That makes the
output safe to assert against a golden file in an automated test, and
reproducible for content-addressed builds.

## Scope

- **Text, lines, rectangles.** The standard-14 fonts, solid fills / strokes, and
  RGB colour. No curves / paths beyond rectangles, no clipping, no transparency.
- **No embedded fonts or images** yet - a follow-on. Only the built-in base
  fonts, so no font file is embedded and non-Latin text is out of scope.
- **A writer, not a reader.** It generates PDFs; it does not parse them.

## See also

- [compress.md](../libraries/compress.md) - the FlateDecode (`zlib`) streams.
- [htmlwriter.md](htmlwriter.md) / [label.md](label.md) - the sibling
  format-generation modules.
- [fs.md](../libraries/fs.md) - `writeBytes` to save the rendered PDF.
- [modules/index.md](index.md) - the module catalog and import rules.
