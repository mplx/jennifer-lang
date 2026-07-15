# `influxdb` - InfluxDB time-series client

Import with `import "influxdb.j" as influxdb;`. Write measurements to InfluxDB
(1.x) as line-protocol points, and run InfluxQL queries that come back as parsed
`Series`. Built on the [`http`](http.md) module, so it needs the default
`jennifer` binary. A failed request throws `Error{kind: "influxdb"}`.

```jennifer
import "influxdb.j" as influxdb;

def db as influxdb.Client init influxdb.client("http://localhost:8086", "metrics");
def p as influxdb.Point init influxdb.field(
    influxdb.tag(influxdb.point("cpu"), "host", "server01"), "value", 0.64);
influxdb.write($db, [$p]);

def r as influxdb.Result init influxdb.query($db, "SELECT last(\"value\") FROM cpu");
```

Runnable: [`examples/modules/influxdb_demo.j`](https://github.com/mplx/jennifer-lang/blob/main/examples/modules/influxdb_demo.j).

## Client

```jennifer
def struct influxdb.Client {
    url as string,        # base URL, e.g. "http://localhost:8086"
    db as string,         # database name
    user as string,       # username ("" for no auth)
    password as string    # password
};
```

| Call | Returns | |
| ---- | ------- | - |
| `influxdb.client(url, db)` | `Client` | connect to a database, no authentication |
| `influxdb.clientWith(url, db, user, password)` | `Client` | with HTTP Basic-auth credentials |

## Writing points

A `Point` is built with value-semantic builders - each returns a fresh `Point`,
so they chain. Field types are carried as pre-rendered line-protocol fragments,
so one point can mix float, integer, string, and boolean fields (Jennifer maps
are homogeneous, so a single typed map could not).

| Call | Returns | |
| ---- | ------- | - |
| `influxdb.point(measurement)` | `Point` | start a point (no tags/fields yet) |
| `influxdb.tag(p, key, value)` | `Point` | add an indexed string tag |
| `influxdb.field(p, key, value)` | `Point` | add a **float** field |
| `influxdb.intField(p, key, value)` | `Point` | add an **int** field (line-protocol `i` suffix) |
| `influxdb.stringField(p, key, value)` | `Point` | add a **string** field (quoted, escaped) |
| `influxdb.boolField(p, key, value)` | `Point` | add a **bool** field |
| `influxdb.at(p, unixNanos)` | `Point` | set an explicit timestamp (nanoseconds) |
| `influxdb.atTime(p, t)` | `Point` | set the timestamp from a `time.Time` |
| `influxdb.line(p)` | `string` | render one line-protocol line (throws if no fields) |
| `influxdb.write(c, points)` | | write a `list of Point` to the database |

Line-protocol escaping is automatic: measurement names escape space and comma;
tag keys/values and field keys escape space, comma, and `=`; string field
values are double-quoted with `"` and `\` escaped. A point with **no fields**
is invalid line protocol, so `line` / `write` throw for one. `write` posts to
`/write?db=...&precision=ns` and throws on a non-2xx response, surfacing the
server's `{"error": ...}` message when present.

```jennifer
def p as influxdb.Point init influxdb.point("cpu");
$p = influxdb.tag($p, "host", "server01");
$p = influxdb.field($p, "value", 0.64);
$p = influxdb.intField($p, "cores", 8);
influxdb.line($p);   # cpu,host=server01 value=0.64,cores=8i
```

## Querying

```jennifer
def struct influxdb.Series {
    name as string,                       # measurement name
    tags as map of string to string,      # GROUP BY tag set ({} if none)
    columns as list of string,            # column names, e.g. ["time", "value"]
    values as list of list of string      # rows, one stringified cell per column
};
def struct influxdb.Result {
    series as list of Series              # flattened across every statement
};
```

`influxdb.query(client, influxql)` runs an InfluxQL statement against `/query`
and parses the tabular JSON into `Series` (the same read-a-parsed-result shape
as [`prometheus`](prometheus.md)'s retrieval half). Every **cell is
stringified** - `time` comes back as its RFC 3339 string, numbers via their
shortest form, booleans as `"true"` / `"false"`, and JSON `null` as `""` - so a
homogeneous `list of list of string` can hold a row of otherwise mixed-type
columns. Convert a cell you know is numeric with `convert.toFloat`. A
per-statement `error` in the response throws `Error{kind: "influxdb"}`.

```jennifer
def r as influxdb.Result init influxdb.query($db, "SELECT value FROM cpu");
for (def s in $r.series) {
    for (def row in $s.values) {
        # row[0] = time (string), row[1] = value (stringified number)
    }
}
```

## Scope

- **InfluxDB 1.x line protocol + InfluxQL.** The 2.x Flux API and the
  `/api/v2/write` org/bucket model are not covered; a v2 backend would be a
  second selectable backend under stance 1, added on a concrete need.
- **Nanosecond write precision** (`precision=ns`); a point's timestamp is an
  integer nanosecond value (or a `time.Time` via `atTime`).
- **Stringified query cells.** The result keeps rows homogeneous (`list of list
  of string`) rather than exposing a typed cell union; you convert numeric
  columns yourself.
- **Basic auth only.** Token auth and TLS client certs are not wired; use a
  reverse proxy for those, or an unauthenticated local endpoint.

## See also

- [http.md](http.md) - the HTTP client this module builds on.
- [prometheus.md](prometheus.md) - the pull-based metrics sibling; its retrieval
  half shares this parsed-result shape.
- [modules/index.md](index.md) - the module catalog and import rules.
