# SPDX-License-Identifier: LGPL-3.0-only
# Copyright (C) 2026 <developer@mplx.eu>
#
# ansi.j - terminal styling as explicit string wrappers. The first module
# built on Jennifer's module system: pure Jennifer, no Go. Colour is gated on
# stdout being a terminal (with the NO_COLOR / FORCE_COLOR override), so
# wrapped output stays clean when it is redirected to a file or a pipe.
#
#     import "ansi.j" as ansi;
#     io.printf("%s\n", ansi.bold(ansi.red("error")));
use os;
use maps;
use convert;
use regex;

# The ESC control byte (27) has no string-literal escape in Jennifer, so it is
# built from a one-byte `bytes`. One SGR sequence is ESC + "[" + code + "m".
def const ESC as string init makeEsc();

func makeEsc() {
    def b as bytes;
    $b[] = 27;
    return convert.stringFromBytes($b, "utf-8");
}

# SGR codes for foreground colour, background colour, and text style.
def const FG as map of string to string init {
    "black": "30", "red": "31", "green": "32", "yellow": "33",
    "blue": "34", "magenta": "35", "cyan": "36", "white": "37",
    "gray": "90", "grey": "90"
};
def const BG as map of string to string init {
    "black": "40", "red": "41", "green": "42", "yellow": "43",
    "blue": "44", "magenta": "45", "cyan": "46", "white": "47"
};
def const STYLE as map of string to string init {
    "bold": "1", "dim": "2", "italic": "3", "underline": "4", "reverse": "7"
};

# enabled reports whether to emit escapes at all: NO_COLOR forces off,
# FORCE_COLOR forces on, otherwise gate on stdout being a terminal (and
# default on when the host cannot tell). Stateless - re-read every call, so
# there is no toggle to store (a module holds no mutable state).
func enabled() {
    if (len(os.getEnv("NO_COLOR")) > 0) {
        return false;
    }
    if (len(os.getEnv("FORCE_COLOR")) > 0) {
        return true;
    }
    return os.isTerminal("stdout");
}

# wrap puts one SGR code around s, then resets - but only when colour is on.
func wrap(s as string, code as string) {
    if (not enabled()) {
        return $s;
    }
    return ESC + "[" + $code + "m" + $s + ESC + "[0m";
}

# lookup returns the SGR code for name in table, or throws on an unknown name.
func lookup(table as map of string to string, name as string, kind as string) {
    if (not maps.has($table, $name)) {
        throw Error{kind: "value", message: "unknown ansi " + $kind + ": " + $name, file: "", line: 0, col: 0};
    }
    return $table[$name];
}

# color / bgColor / style wrap s in the named colour or style; rgb takes a
# 24-bit truecolor foreground.
export func color(s as string, name as string) {
    return wrap($s, lookup(FG, $name, "colour"));
}
export func bgColor(s as string, name as string) {
    return wrap($s, lookup(BG, $name, "background"));
}
export func style(s as string, name as string) {
    return wrap($s, lookup(STYLE, $name, "style"));
}
export func rgb(s as string, r as int, g as int, b as int) {
    return wrap($s, "38;2;" + convert.toString($r) + ";" + convert.toString($g) + ";" + convert.toString($b));
}

# strip removes every SGR escape - the inverse of the wrappers, regardless of
# whether colour is currently enabled.
export func strip(s as string) {
    return regex.replace(ESC + "\\[[0-9;]*m", $s, "");
}

# Convenience shortcuts for the common colours and styles.
export func black(s as string) { return color($s, "black"); }
export func red(s as string) { return color($s, "red"); }
export func green(s as string) { return color($s, "green"); }
export func yellow(s as string) { return color($s, "yellow"); }
export func blue(s as string) { return color($s, "blue"); }
export func magenta(s as string) { return color($s, "magenta"); }
export func cyan(s as string) { return color($s, "cyan"); }
export func white(s as string) { return color($s, "white"); }
export func gray(s as string) { return color($s, "gray"); }
export func bold(s as string) { return style($s, "bold"); }
export func dim(s as string) { return style($s, "dim"); }
export func italic(s as string) { return style($s, "italic"); }
export func underline(s as string) { return style($s, "underline"); }
export func reverse(s as string) { return style($s, "reverse"); }
