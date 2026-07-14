# `ical` - iCalendar (RFC 5545) build and parse

Import with `import "ical.j" as ical;`. Build a calendar of events and encode it
to **iCalendar** text (a `VCALENDAR` of `VEVENT`s), and parse that text back into
a `Calendar`. Pure Jennifer over `strings` / `lists` + `time` - no Go engine, so
it runs on **both** binaries.

```jennifer
import "ical.j" as ical;
use time;

def ev as ical.Event init ical.event(
    "launch@team",
    time.fromIso("2024-06-20T14:00:00Z"),
    time.fromIso("2024-06-20T15:30:00Z"),
    "Product launch");
def cal as ical.Calendar init ical.add(ical.calendar(), $ev);
def text as string init ical.encode($cal);   # BEGIN:VCALENDAR ... END:VCALENDAR
```

Runnable: [`examples/modules/ical_demo.j`](https://github.com/mplx/jennifer-lang/blob/main/examples/modules/ical_demo.j).

## Types

Both structs have public fields (read them directly - `$cal.events`,
`$ev.summary`); the builder functions are the conventional way to construct them.

```jennifer
def struct ical.Calendar { prodid as string, events as list of Event };
def struct ical.Event {
    uid as string,
    stamp as time.Time,      # DTSTAMP
    start as time.Time,      # DTSTART
    end as time.Time,        # DTEND
    summary as string,       # SUMMARY
    description as string,   # DESCRIPTION ("" when unset)
    location as string       # LOCATION ("" when unset)
};
```

## Building

| Call | Returns | |
| ---- | ------- | - |
| `ical.calendar()` | `Calendar` | an empty calendar with the default `PRODID` |
| `ical.calendarWith(prodid)` | `Calendar` | an empty calendar with a custom `PRODID` |
| `ical.event(uid, start, end, summary)` | `Event` | an event; `DTSTAMP` defaults to `start` |
| `ical.describe(ev, description)` | `Event` | a copy with the description set |
| `ical.locate(ev, location)` | `Event` | a copy with the location set |
| `ical.add(cal, ev)` | `Calendar` | a copy with the event appended |

The builders are **value-semantic** - `describe` / `locate` / `add` return a
fresh copy and never mutate their argument, so you thread them:

```jennifer
def ev as ical.Event init ical.event("id", $start, $end, "Meeting");
$ev = ical.describe($ev, "agenda...");
$ev = ical.locate($ev, "Room 5");
def cal as ical.Calendar init ical.add(ical.calendar(), $ev);
```

## Encoding and parsing

| Call | Returns | |
| ---- | ------- | - |
| `ical.encode(cal)` | `string` | the calendar as RFC 5545 text (CRLF-terminated) |
| `ical.parse(text)` | `Calendar` | parse iCalendar text back into a calendar |

`parse(encode(cal))` round-trips the data. `encode` writes CRLF line endings,
escapes text values, folds long lines, and emits `DESCRIPTION` / `LOCATION` only
when non-empty. `parse` unfolds folded lines, ignores property parameters (the
`;KEY=VALUE` after a name, e.g. `DTSTART;VALUE=DATE-TIME`), unescapes text,
skips a `VEVENT` with no `DTSTART`, and defaults a missing `DTEND` to the start.

## Dates and times

`DTSTAMP` / `DTSTART` / `DTEND` go through `time`. `encode` writes each as a UTC
`DATE-TIME` (`20240615T130000Z`), normalising a non-UTC `time.Time` to UTC first,
so the output is always a correct `Z` value. `parse` accepts the UTC `...Z` form,
a floating `DATE-TIME` (no `Z`, read as UTC), and a bare `DATE` (`20240615`).

## Text escaping and folding

Text values (`SUMMARY` / `DESCRIPTION` / `LOCATION` / `UID` / `PRODID`) use RFC
5545 escaping: a backslash, `;`, `,`, and any newline become `\\`, `\;`, `\,`,
and `\n`. Content lines longer than 75 characters are folded onto continuation
lines (a CRLF followed by a space), and `parse` rejoins them - so a long
description survives the round-trip intact.

## Scope

- **`VEVENT` only.** No `VTODO` / `VJOURNAL` / `VALARM` / `VTIMEZONE`, no
  recurrence rules (`RRULE`), no attendees / organizer. A focused calendar-of-
  events surface; the escaping / folding / date discipline is the reusable core.
- **UTC date-times.** Events are stored and emitted in UTC. There is no
  per-event `TZID` timezone reference (the `time` library ships fixed-offset
  zones only); a `TZID` parameter on input is ignored and the value read as-is.
- **Fold width in characters.** Long lines fold on rune boundaries at 75
  characters (never splitting a multi-byte character), rather than strictly on
  75 octets - valid output that every reader unfolds.

## See also

- [time.md](../libraries/time.md) - the instant / duration types the dates use.
- [strings.md](../libraries/strings.md) - the text surface the codec is built on.
- [modules/index.md](index.md) - the module catalog and import rules.
