# `font` - TrueType / SFNT font parsing

Import with `import "font.j" as font;`. A **pure-Jennifer** TrueType parser: read
a `.ttf` from `bytes` and expose its metrics, character map, and glyph outlines -
no Go, no dependency, just the `bytes` type and the bitwise operators for the
big-endian tables, so it runs on **both binaries**.

Named `font` (not `truetype`) because the SFNT container also carries CFF /
PostScript outlines: this ships the **TrueType `glyf` backend**, and a CFF
backend can be added later, detected on parse. It parses the core tables - `head`,
`cmap` (formats 4 and 12), `maxp` / `hhea` / `hmtx`, `loca` / `glyf` (simple
**and** composite glyphs, quadratic curves), and `name` - enough to lay out and
outline a string.

```jennifer
use io;
import "font.j" as font;

def f as font.Font init font.open("/usr/share/fonts/TTF/DejaVuSans.ttf");
io.printf("%s, %d upem\n", font.name($f), font.unitsPerEm($f));
io.printf("<path d=\"%s\"/>\n", font.glyphPath($f, 65));   # outline of 'A'
```

Runnable: [`examples/modules/font_demo.j`](https://github.com/jennifer-language/jennifer/blob/main/examples/modules/font_demo.j) (a word rendered to an SVG).

## Surface

| Call | Returns | Notes |
| ---- | ------- | ----- |
| `font.parse(b)` | `Font` | Parse a font from `bytes`. |
| `font.open(path)` | `Font` | Read a `.ttf` file and parse it. |
| `font.unitsPerEm(f)` | `int` | The em-square size - the coordinate scale of every metric and outline. |
| `font.name(f)` | `string` | The font family name. |
| `font.advance(f, codepoint)` | `int` | The horizontal advance width of a codepoint's glyph, in font units. |
| `font.glyphPath(f, codepoint)` | `string` | The glyph outline as an SVG path `d` string. |
| `font.glyph(f, codepoint)` | `Glyph` | The raw outline: contours of on / off-curve points, advance, and bounding box. |

A codepoint the font lacks maps to glyph 0 (`.notdef`).

## Coordinates and outlines

Everything is in **font units** - divide by `unitsPerEm` (typically 1000 or
2048) to get em fractions, then multiply by your point size. Fonts store **y
pointing up**, so flip y for screen rendering (an SVG `transform="scale(1,-1)"`
around the whole word does it).

`font.glyphPath` emits an SVG path: `M` / `L` for straight segments and `Q`
(quadratic Bezier) for curves - TrueType outlines are quadratic. For the raw
outline, `font.glyph` returns a `Glyph`:

- `Glyph { advance, xMin, yMin, xMax, yMax, contours as list of Contour }`
- `Contour { points as list of Point }`
- `Point { x, y, onCurve }` - an off-curve point is a quadratic control point.

Composite glyphs (an accented letter built from a base plus a mark) are resolved
into a single set of contours, with each component's translation and scale
applied.

## Laying out a string

Walk the string, outline each glyph, and advance the pen by its advance width:

```jennifer
def x as int init 0;
for (def i as int init 0; $i < len($chars); $i = $i + 1) {
    def cp as int init charCode($chars[$i]);
    render(font.glyphPath($f, $cp), $x);              # your renderer, offset by x
    $x = $x + font.advance($f, $cp);
}
```

Kerning and complex shaping (GPOS / GSUB) are out of scope - advance-width
layout only.

## Scope

**In:** TrueType `glyf` outlines (simple + composite, quadratic curves), `cmap`
formats 4 and 12, `hmtx` advances, and the family name. **Out of v1:** CFF /
PostScript outlines (cubic curves - the second backend), hinting, kerning and
shaping beyond `hmtx`, colour / emoji tables, and variable-font axes.

The tests parse a tiny committed fixture (`modules/testdata/font_fixture.ttf`,
regenerable with `scripts/gen-font-fixture.py`); the parser is also exercised
against real system fonts.
