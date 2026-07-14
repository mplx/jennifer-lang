# SPDX-License-Identifier: LGPL-3.0-only
# Copyright (C) 2026 <developer@mplx.eu>

/**
 * JSON Lines (JSONL / NDJSON): newline-delimited JSON, one independent value per
 * line. A thin framing layer over `json` - each record is a `json.Value`, so
 * `encode` / `decode` compose `json.encode` / `json.decode` with a `\n` split /
 * join, and the file helpers add `fs`. Pure Jennifer; both binaries.
 *
 * `encode(records)` writes one compact JSON value per line (trailing newline);
 * `decode(text)` parses each non-blank line back into a `json.Value`, so
 * `decode(encode(records))` round-trips. Whole-file `readFile` / `writeFile` /
 * `appendFile` cover the common case; a streaming `Reader` reads one record at a
 * time for files too large to hold in memory.
 * @module jsonl
 * @example
 * import "jsonl.j" as jsonl;
 * use json;
 * def rows as list of json.Value init [json.decode("{\"a\":1}"), json.decode("[2,3]")];
 * def text as string init jsonl.encode($rows);   # {"a":1}\n[2,3]\n
 * def back as list of json.Value init jsonl.decode($text);
 */
use json;
use strings;
use fs;

# --- in-memory (exported) ---------------------------------------------------

/**
 * Encode records as JSONL: one compact JSON value per line, each terminated by a
 * newline. An empty list yields the empty string.
 * @param records {list of json.Value} the records to encode
 * @return {string} the JSONL text
 */
export func encode(records as list of json.Value) {
    def out as string init "";
    for (def rec in $records) {
        $out = $out + json.encode($rec) + "\n";
    }
    return $out;
}

/**
 * Decode JSONL text into records, one `json.Value` per non-blank line. Blank and
 * whitespace-only lines are skipped, and a trailing `\r` (CRLF input) is
 * trimmed, so `decode(encode(records))` round-trips.
 * @param text {string} the JSONL text
 * @return {list of json.Value} the parsed records
 */
export func decode(text as string) {
    def records as list of json.Value init [];
    for (def raw in strings.split($text, "\n")) {
        def line as string init strings.trim($raw);
        if (not ($line == "")) {
            $records[] = json.decode($line);
        }
    }
    return $records;
}

# --- whole-file (exported) --------------------------------------------------

/**
 * Read and decode a whole JSONL file.
 * @param path {string} the file path
 * @return {list of json.Value} the records
 */
export func readFile(path as string) {
    return decode(fs.readString($path));
}

/**
 * Encode records and write them to a file, replacing any existing content.
 * @param path {string} the file path
 * @param records {list of json.Value} the records to write
 */
export func writeFile(path as string, records as list of json.Value) {
    fs.writeString($path, encode($records));
    return null;
}

/**
 * Encode records and append them to a file (created if missing) - the common
 * JSONL pattern of adding rows to a growing log.
 * @param path {string} the file path
 * @param records {list of json.Value} the records to append
 */
export func appendFile(path as string, records as list of json.Value) {
    fs.appendString($path, encode($records));
    return null;
}

# --- streaming reader (exported) --------------------------------------------

/**
 * A line-at-a-time reader over an open file, for JSONL too large to hold in
 * memory. Build with `openReader`; the wrapped `fs.File` shares its read
 * position across value copies (a handle), so successive `readRecord` calls
 * advance the same stream.
 * @field file {fs.File} the underlying open file handle
 */
export def struct Reader {
    file as fs.File
};

/**
 * Open a JSONL file for streaming.
 * @param path {string} the file path
 * @return {Reader} the reader
 */
export func openReader(path as string) {
    return Reader{ file: fs.open($path, "read") };
}

/**
 * Whether the reader has more input (false once the file is exhausted). Guard
 * `readRecord` with this.
 * @param reader {Reader} the reader
 * @return {bool} true if a further read may return a record
 */
export func hasMore(reader as Reader) {
    return not fs.eof($reader.file);
}

/**
 * Read and decode the next record, skipping blank lines. Mirrors `fs.readLine`:
 * it throws `Error{kind: "jsonl"}` when the stream is exhausted, so guard it with
 * `hasMore`.
 * @param reader {Reader} the reader
 * @return {json.Value} the next record
 * @throws {Error} kind "jsonl" when there is no further record
 */
export func readRecord(reader as Reader) {
    while (not fs.eof($reader.file)) {
        def line as string init strings.trim(fs.readLine($reader.file));
        if (not ($line == "")) {
            return json.decode($line);
        }
    }
    throw Error{ kind: "jsonl", message: "readRecord: no more records (guard with hasMore)", file: "", line: 0, col: 0 };
}

/**
 * Close a streaming reader.
 * @param reader {Reader} the reader
 */
export func closeReader(reader as Reader) {
    fs.close($reader.file);
    return null;
}
