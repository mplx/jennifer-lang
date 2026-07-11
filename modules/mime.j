# SPDX-License-Identifier: LGPL-3.0-only
# Copyright (C) 2026 <developer@mplx.eu>
#
# mime.j - build and parse MIME messages (RFC 5322 headers + RFC 2045/2046
# bodies): a header-and-boundary structure that is exactly the text
# orchestration a `.j` module does well. The transfer codecs (base64,
# quoted-printable) are delegated to the `encoding` system library. This is the
# message-structure foundation the `mail` clients (SMTP / POP3 / IMAP) build
# on; it does no networking itself.
#
#     import "mime.j" as mime;
#     def msg as mime.Part init mime.text("text/plain", "Hello, monde: café.");
#     $msg = mime.withHeader($msg, "Subject", "Hi");
#     io.printf("%s", mime.encode($msg));
#     def back as mime.Part init mime.parse(mime.encode($msg));
#
# A `Part` is a leaf (headers + a decoded-text `body` with a transfer
# `encoding`) or a multipart container (headers + child `parts` + a
# `boundary`). Bodies are held decoded as text; encode applies the transfer
# encoding, parse removes it. Content is text (UTF-8): binary attachments (a
# `bytes` body) and RFC 2047 encoded-words for non-ASCII headers are not yet
# handled.
use strings;
use convert;
use encoding;

# A single header field.
export def struct Header {
    name as string,
    value as string
};

# A MIME part: a leaf (`body` + `encoding`, `parts` empty) or a multipart
# container (`parts` + `boundary`, `body` empty).
export def struct Part {
    headers as list of Header,
    body as string,
    encoding as string,
    parts as list of Part,
    boundary as string
};

# --- small text helpers (private) ----------------------------------

func mkHeader(name as string, value as string) {
    return Header{name: $name, value: $value};
}

# crlf normalises any line endings to CRLF (canonical for a MIME message).
func crlf(s as string) {
    def out as string init strings.replace($s, "\r\n", "\n");
    return strings.replace($out, "\n", "\r\n");
}

# stripCR drops a single trailing CR (a line split off a CRLF stream).
func stripCR(line as string) {
    if (strings.endsWith($line, "\r")) {
        return strings.substring($line, 0, len($line) - 1);
    }
    return $line;
}

# stripWS removes all spaces and line breaks (base64 tolerates none on decode).
func stripWS(s as string) {
    def out as string init strings.replace($s, "\r", "");
    $out = strings.replace($out, "\n", "");
    return strings.replace($out, " ", "");
}

# wrapLines folds a long unbroken string (base64) into 76-column CRLF lines.
func wrapLines(s as string) {
    def out as string init "";
    def cs as list of string init strings.chars($s);
    def n as int init len($cs);
    def i as int init 0;
    while ($i < $n) {
        if ($i > 0 and $i % 76 == 0) {
            $out = $out + "\r\n";
        }
        $out = $out + $cs[$i];
        $i = $i + 1;
    }
    return $out;
}

# --- transfer encoding (private) -----------------------------------

func encodeBody(body as string, enc as string) {
    def b as bytes init convert.bytesFromString($body, "utf-8");
    if ($enc == "base64") {
        return wrapLines(encoding.toText($b, "base64"));
    }
    if ($enc == "quoted-printable") {
        return crlf(encoding.toText($b, "quoted-printable"));
    }
    return crlf($body);
}

func decodeBody(raw as string, enc as string) {
    if ($enc == "base64") {
        return convert.stringFromBytes(encoding.fromText(stripWS($raw), "base64"), "utf-8");
    }
    if ($enc == "quoted-printable") {
        return convert.stringFromBytes(encoding.fromText($raw, "quoted-printable"), "utf-8");
    }
    return $raw;
}

# --- header lookup (private) ---------------------------------------

# findHeader returns the value of the named header (case-insensitive) or "".
func findHeader(headers as list of Header, name as string) {
    def lname as string init strings.lower($name);
    for (def h in $headers) {
        if (strings.lower($h.name) == $lname) {
            return $h.value;
        }
    }
    return "";
}

# setHeaderIn returns the header list with `name` set (replaced if present,
# else appended).
func setHeaderIn(headers as list of Header, name as string, value as string) {
    def out as list of Header init [];
    def lname as string init strings.lower($name);
    def found as bool init false;
    for (def h in $headers) {
        if (strings.lower($h.name) == $lname) {
            $out[] = mkHeader($h.name, $value);
            $found = true;
        } else {
            $out[] = $h;
        }
    }
    if (not $found) {
        $out[] = mkHeader($name, $value);
    }
    return $out;
}

# --- content-type parsing (private) --------------------------------

# typeOnly is the media type without parameters ("text/plain" from
# "text/plain; charset=utf-8").
func typeOnly(ct as string) {
    def semi as int init strings.indexOf($ct, ";");
    if ($semi >= 0) {
        return strings.trim(strings.substring($ct, 0, $semi));
    }
    return strings.trim($ct);
}

# extractBoundary reads the `boundary=` parameter of a Content-Type, quoted or
# bare, or "" when absent.
func extractBoundary(ct as string) {
    def idx as int init strings.indexOf(strings.lower($ct), "boundary=");
    if ($idx < 0) {
        return "";
    }
    def rest as string init strings.trim(strings.substring($ct, $idx + 9, len($ct)));
    if (strings.startsWith($rest, "\"")) {
        def tail as string init strings.substring($rest, 1, len($rest));
        def end as int init strings.indexOf($tail, "\"");
        if ($end >= 0) {
            return strings.substring($tail, 0, $end);
        }
        return "";
    }
    def semi as int init strings.indexOf($rest, ";");
    if ($semi >= 0) {
        return strings.trim(strings.substring($rest, 0, $semi));
    }
    return $rest;
}

# --- header parsing (private) --------------------------------------

# splitHeaderLine parses one unfolded `Name: value` line.
func splitHeaderLine(line as string) {
    def idx as int init strings.indexOf($line, ":");
    if ($idx < 0) {
        return mkHeader(strings.trim($line), "");
    }
    def name as string init strings.trim(strings.substring($line, 0, $idx));
    def value as string init strings.trim(strings.substring($line, $idx + 1, len($line)));
    return mkHeader($name, $value);
}

# parseHeaders parses a header block, unfolding continuation lines (those that
# begin with whitespace) into the field above.
func parseHeaders(text as string) {
    def hs as list of Header init [];
    def cur as string init "";
    for (def line in strings.split($text, "\n")) {
        def l as string init stripCR($line);
        def folds as bool init strings.startsWith($l, " ") or strings.startsWith($l, "\t");
        if (len($l) > 0 and $folds) {
            $cur = $cur + " " + strings.trim($l);
        } else {
            if (len($cur) > 0) {
                $hs[] = splitHeaderLine($cur);
            }
            $cur = $l;
        }
    }
    if (len($cur) > 0) {
        $hs[] = splitHeaderLine($cur);
    }
    return $hs;
}

# parseMultipart splits a multipart body on its boundary and parses each
# body-part; the preamble before the first boundary and epilogue after the
# close delimiter are dropped.
func parseMultipart(body as string, boundary as string) {
    def delim as string init "--" + $boundary;
    def close as string init $delim + "--";
    def parts as list of Part init [];
    def cur as list of string init [];
    def collecting as bool init false;
    for (def line in strings.split($body, "\n")) {
        def t as string init stripCR($line);
        if ($t == $delim or $t == $close) {
            if ($collecting) {
                $parts[] = parse(strings.join($cur, "\n"));
            }
            $cur = [];
            $collecting = not ($t == $close);
        } else {
            if ($collecting) {
                $cur[] = $line;
            }
        }
    }
    return $parts;
}

# --- building (exported) -------------------------------------------

# text builds a leaf text part; the body is sent 7bit when ASCII, else
# quoted-printable. `charset=utf-8` is appended to the content type.
export func text(contentType as string, body as string) {
    def enc as string init "7bit";
    if (not encoding.isAscii(convert.bytesFromString($body, "utf-8"))) {
        $enc = "quoted-printable";
    }
    def hs as list of Header init [];
    $hs[] = mkHeader("Content-Type", $contentType + "; charset=utf-8");
    $hs[] = mkHeader("Content-Transfer-Encoding", $enc);
    return Part{headers: $hs, body: $body, encoding: $enc, parts: [], boundary: ""};
}

# attachment builds a base64 leaf part with a filename. `body` is text content
# (UTF-8); binary bodies are not yet supported.
export func attachment(filename as string, contentType as string, body as string) {
    def hs as list of Header init [];
    $hs[] = mkHeader("Content-Type", $contentType);
    $hs[] = mkHeader("Content-Transfer-Encoding", "base64");
    $hs[] = mkHeader("Content-Disposition", "attachment; filename=\"" + $filename + "\"");
    return Part{headers: $hs, body: $body, encoding: "base64", parts: [], boundary: ""};
}

# multipart builds a container part; every child ships under one `boundary`.
export func multipart(subtype as string, boundary as string, parts as list of Part) {
    def hs as list of Header init [];
    $hs[] = mkHeader("Content-Type", "multipart/" + $subtype + "; boundary=\"" + $boundary + "\"");
    return Part{headers: $hs, body: "", encoding: "", parts: $parts, boundary: $boundary};
}

# withHeader returns a copy of the part with `name` set (replaced or appended).
export func withHeader(part as Part, name as string, value as string) {
    def np as Part init $part;
    $np.headers = setHeaderIn($part.headers, $name, $value);
    return $np;
}

# --- encoding (exported) -------------------------------------------

func encodeMultipart(p as Part) {
    def out as string init "";
    for (def sub in $p.parts) {
        $out = $out + "--" + $p.boundary + "\r\n" + encode($sub) + "\r\n";
    }
    return $out + "--" + $p.boundary + "--\r\n";
}

# encode serializes a part (and its subtree) to a MIME message string with CRLF
# line endings and the declared transfer encodings applied.
export func encode(part as Part) {
    def out as string init "";
    for (def h in $part.headers) {
        $out = $out + $h.name + ": " + $h.value + "\r\n";
    }
    $out = $out + "\r\n";
    if (len($part.boundary) > 0) {
        return $out + encodeMultipart($part);
    }
    return $out + encodeBody($part.body, $part.encoding);
}

# --- parsing (exported) --------------------------------------------

# parse reads a MIME message string into a Part tree: headers are unfolded, a
# multipart body is split on its boundary (recursively), and a leaf body is
# transfer-decoded.
export func parse(text as string) {
    def norm as string init strings.replace($text, "\r\n", "\n");
    def idx as int init strings.indexOf($norm, "\n\n");
    def headerText as string init $norm;
    def bodyText as string init "";
    if ($idx >= 0) {
        $headerText = strings.substring($norm, 0, $idx);
        $bodyText = strings.substring($norm, $idx + 2, len($norm));
    }
    def hs as list of Header init parseHeaders($headerText);
    def boundary as string init extractBoundary(findHeader($hs, "Content-Type"));
    if (len($boundary) > 0) {
        def kids as list of Part init parseMultipart($bodyText, $boundary);
        return Part{headers: $hs, body: "", encoding: "", parts: $kids, boundary: $boundary};
    }
    def enc as string init findHeader($hs, "Content-Transfer-Encoding");
    if (len($enc) == 0) {
        $enc = "7bit";
    }
    return Part{headers: $hs, body: decodeBody($bodyText, $enc), encoding: $enc,
        parts: [], boundary: ""};
}

# --- accessors + helpers (exported) --------------------------------

# headerValue returns a part's header value (case-insensitive) or "".
export func headerValue(part as Part, name as string) {
    return findHeader($part.headers, $name);
}

# body returns a leaf part's decoded text body.
export func body(part as Part) {
    return $part.body;
}

# parts returns a multipart container's child parts.
export func parts(part as Part) {
    return $part.parts;
}

# contentType returns a part's media type without parameters (e.g. "text/plain").
export func contentType(part as Part) {
    return typeOnly(findHeader($part.headers, "Content-Type"));
}

# needsQuoting reports whether a display name must be a quoted-string.
func needsQuoting(s as string) {
    if (strings.contains($s, ",") or strings.contains($s, "<")) {
        return true;
    }
    return strings.contains($s, "@") or strings.contains($s, "\"");
}

# address formats an RFC 5322 mailbox: `email` alone, or `Name <email>` with
# the name quoted when it carries a special character.
export func address(name as string, email as string) {
    if (len($name) == 0) {
        return $email;
    }
    def display as string init $name;
    if (needsQuoting($name)) {
        $display = "\"" + strings.replace($name, "\"", "\\\"") + "\"";
    }
    return $display + " <" + $email + ">";
}
