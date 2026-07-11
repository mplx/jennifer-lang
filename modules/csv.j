# SPDX-License-Identifier: LGPL-3.0-only
# Copyright (C) 2026 <developer@mplx.eu>
#
# csv.j - RFC 4180 comma-separated values: parse text into rows of fields and
# format rows back into text, with a quoting-aware hand-written scanner. Pure
# Jennifer - no Go, no system library. The delimiter is configurable, so the
# same code reads and writes TSV and other single-character-separated formats.
#
#     import "csv.j" as csv;
#     def rows as list of list of string init csv.parse("a,b\n1,\"x,y\"");
#     def recs as list of map of string to string init csv.toRecords($rows);
#
# Records separate on LF or CRLF (a bare CR outside quotes also ends a record);
# a field is quoted with `"` when it contains the delimiter, a quote, or a
# newline, and an embedded quote doubles to `""`. format() joins records with
# LF and adds no trailing newline, so parse(format(rows)) round-trips the data.
use strings;
use maps;

# --- quoting helpers (private) -------------------------------------

# needsQuote reports whether a field must be wrapped in quotes: it carries the
# delimiter, a quote, or a line break.
func needsQuote(f as string, delim as string) {
    if (strings.contains($f, $delim)) {
        return true;
    }
    if (strings.contains($f, '"')) {
        return true;
    }
    if (strings.contains($f, "\n") or strings.contains($f, "\r")) {
        return true;
    }
    return false;
}

# quoteField returns a field ready to write: wrapped in quotes with any
# embedded quote doubled when quoting is required, otherwise unchanged.
func quoteField(f as string, delim as string) {
    if (needsQuote($f, $delim)) {
        return '"' + strings.replace($f, '"', '""') + '"';
    }
    return $f;
}

# --- parse (exported) ----------------------------------------------

# parseWith scans CSV text with a single-character delimiter and returns its
# rows, each a list of string fields. A quoted field may span the delimiter,
# newlines, and doubled quotes; an empty input yields no rows.
export func parseWith(s as string, delim as string) {
    def rows as list of list of string init [];
    def row as list of string init [];
    def field as string init "";
    def inQuotes as bool init false;
    def fieldStarted as bool init false;
    def rowStarted as bool init false;
    def chars as list of string init strings.chars($s);
    def n as int init len($chars);
    def i as int init 0;
    # Flat guard-and-continue dispatch keeps the scanner readable and shallow.
    while ($i < $n) {
        def c as string init $chars[$i];
        # Inside a quoted field: a doubled quote is a literal, a lone quote
        # closes, anything else is content (delimiters and newlines included).
        if ($inQuotes and $c == '"' and $i + 1 < $n and $chars[$i + 1] == '"') {
            $field = $field + '"';
            $i = $i + 2;
            continue;
        }
        if ($inQuotes and $c == '"') {
            $inQuotes = false;
            $i = $i + 1;
            continue;
        }
        if ($inQuotes) {
            $field = $field + $c;
            $i = $i + 1;
            continue;
        }
        # Outside quotes.
        if ($c == '"') {
            $inQuotes = true;
            $fieldStarted = true;
            $rowStarted = true;
            $i = $i + 1;
            continue;
        }
        if ($c == $delim) {
            $row[] = $field;
            $field = "";
            $fieldStarted = false;
            $rowStarted = true;
            $i = $i + 1;
            continue;
        }
        if ($c == "\n" or $c == "\r") {
            $row[] = $field;
            $rows[] = $row;
            $field = "";
            $row = [];
            $fieldStarted = false;
            $rowStarted = false;
            # Consume the LF of a CRLF pair as one record separator.
            if ($c == "\r" and $i + 1 < $n and $chars[$i + 1] == "\n") {
                $i = $i + 2;
            } else {
                $i = $i + 1;
            }
            continue;
        }
        $field = $field + $c;
        $fieldStarted = true;
        $rowStarted = true;
        $i = $i + 1;
    }
    # Flush the final record unless the text ended exactly on a separator.
    if ($rowStarted or $fieldStarted) {
        $row[] = $field;
        $rows[] = $row;
    }
    return $rows;
}

# parse scans standard comma-delimited CSV.
export func parse(s as string) {
    return parseWith($s, ",");
}

# --- format (exported) ---------------------------------------------

# formatWith joins rows into text with a single-character delimiter, quoting
# each field as needed. Records are separated by LF with no trailing newline.
export func formatWith(rows as list of list of string, delim as string) {
    def out as string init "";
    def firstRow as bool init true;
    for (def row in $rows) {
        if (not $firstRow) {
            $out = $out + "\n";
        }
        $firstRow = false;
        def firstField as bool init true;
        for (def field in $row) {
            if (not $firstField) {
                $out = $out + $delim;
            }
            $firstField = false;
            $out = $out + quoteField($field, $delim);
        }
    }
    return $out;
}

# format joins rows into standard comma-delimited CSV.
export func format(rows as list of list of string) {
    return formatWith($rows, ",");
}

# --- header records (exported) -------------------------------------

# toRecords treats the first row as a header and maps each later row into a
# `map of string to string` keyed by the header names. Every record carries
# every header key (a short row fills missing fields with ""); fields past the
# header width are dropped. An empty input yields no records.
export func toRecords(rows as list of list of string) {
    def records as list of map of string to string init [];
    if (len($rows) == 0) {
        return $records;
    }
    def header as list of string init $rows[0];
    def r as int init 1;
    while ($r < len($rows)) {
        def row as list of string init $rows[$r];
        def rec as map of string to string init {};
        def c as int init 0;
        while ($c < len($header)) {
            if ($c < len($row)) {
                $rec[$header[$c]] = $row[$c];
            } else {
                $rec[$header[$c]] = "";
            }
            $c = $c + 1;
        }
        $records[] = $rec;
        $r = $r + 1;
    }
    return $records;
}

# fromRecords is the inverse of toRecords: it emits the header row followed by
# one row per record, taking fields in header order (a key absent from a record
# writes ""). The explicit header fixes the column order, which map iteration
# does not.
export func fromRecords(header as list of string, records as list of map of string to string) {
    def rows as list of list of string init [];
    $rows[] = $header;
    for (def rec in $records) {
        def row as list of string init [];
        for (def col in $header) {
            if (maps.has($rec, $col)) {
                $row[] = $rec[$col];
            } else {
                $row[] = "";
            }
        }
        $rows[] = $row;
    }
    return $rows;
}
