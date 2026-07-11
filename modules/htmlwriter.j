# SPDX-License-Identifier: LGPL-3.0-only
# Copyright (C) 2026 <developer@mplx.eu>
#
# htmlwriter.j - build an HTML element tree and render it to a correctly
# escaped HTML5 string. Pure Jennifer over `strings` and `lists` - a *writer*,
# not a parser, so it has no dependency on an XML parser: serialization is a
# handful of string operations. The shared output layer any HTML-emitting
# consumer reuses (a Markdown renderer, a documentation generator, a view
# layer).
#
#     import "htmlwriter.j" as html;
#     def kids as list of html.Node init [];
#     $kids[] = html.text("hi & bye");
#     def p as html.Node init html.element("p", [], $kids);
#     io.printf("%s\n", html.render($p));   # <p>hi &amp; bye</p>
#
# Text nodes are escaped on render (`&` `<` `>`); attribute values also escape
# `"`; a `raw` node passes through verbatim for already-trusted markup. Void
# elements (`br`, `img`, ...) render without a closing tag and drop children.
use strings;
use lists;

# A node is one of three kinds, tagged by `kind`: "element" (tag + attrs +
# children), "text" (escaped content), or "raw" (verbatim content). The
# constructors below are the intended way to build one.
export def struct Node {
    kind as string,
    tag as string,
    attrs as list of Attr,
    children as list of Node,
    text as string
};

export def struct Attr {
    name as string,
    value as string
};

# The HTML5 void elements: they never have a closing tag or children.
def const VOID as list of string init ["area", "base", "br", "col", "embed", "hr", "img", "input", "link", "meta", "param", "source", "track", "wbr"];  # lint-disable: L203

# --- constructors (exported) ---------------------------------------

# element builds an element node from a tag, its attributes, and its children.
# Pass `[]` for either when there are none.
export func element(tag as string, attrs as list of Attr, children as list of Node) {
    return Node{kind: "element", tag: $tag, attrs: $attrs, children: $children, text: ""};
}

# text builds a text node; its content is HTML-escaped on render.
export func text(s as string) {
    return Node{kind: "text", tag: "", attrs: [], children: [], text: $s};
}

# raw builds a node whose content is emitted verbatim - for already-trusted
# markup only, since it is not escaped.
export func raw(s as string) {
    return Node{kind: "raw", tag: "", attrs: [], children: [], text: $s};
}

# attr builds one name/value attribute for an element.
export func attr(name as string, value as string) {
    return Attr{name: $name, value: $value};
}

# --- escaping (private + one public helper) ------------------------

# escape returns s with the text-context HTML metacharacters replaced by
# entities (`&` first, so an escaped entity is not re-escaped). Public because
# escaping a bare string for HTML text is useful without building a node.
export func escape(s as string) {
    def out as string init strings.replace($s, "&", "&amp;");
    $out = strings.replace($out, "<", "&lt;");
    $out = strings.replace($out, ">", "&gt;");
    return $out;
}

# escapeAttr additionally escapes the double quote, since attribute values are
# rendered inside double quotes.
func escapeAttr(s as string) {
    def out as string init escape($s);
    return strings.replace($out, '"', "&quot;");
}

# --- rendering (exported) ------------------------------------------

# isVoid reports whether a tag is an HTML5 void element (case-insensitive).
func isVoid(tag as string) {
    return lists.contains(VOID, strings.lower($tag));
}

# renderAttrs renders a leading-space-separated attribute list.
func renderAttrs(attrs as list of Attr) {
    def out as string init "";
    for (def a in $attrs) {
        $out = $out + " " + $a.name + "=\"" + escapeAttr($a.value) + "\"";
    }
    return $out;
}

# render serializes a node and its subtree to an HTML5 string.
export func render(node as Node) {
    if ($node.kind == "text") {
        return escape($node.text);
    }
    if ($node.kind == "raw") {
        return $node.text;
    }
    def out as string init "<" + $node.tag + renderAttrs($node.attrs);
    if (isVoid($node.tag)) {
        return $out + ">";
    }
    $out = $out + ">";
    for (def child in $node.children) {
        $out = $out + render($child);
    }
    return $out + "</" + $node.tag + ">";
}

# renderAll serializes a list of sibling nodes (a fragment) in order.
export func renderAll(nodes as list of Node) {
    def out as string init "";
    for (def n in $nodes) {
        $out = $out + render($n);
    }
    return $out;
}
