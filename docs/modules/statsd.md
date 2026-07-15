# `statsd` - StatsD metrics client

Import with `import "statsd.j" as statsd;`. Emit `metric:value|type` lines to a
StatsD / Datadog / Telegraf agent over UDP. This is the **push** counterpart to
a pull-based scrape: it is **fire-and-forget** (UDP, no reply, no error when no
agent is listening), so a metric costs one datagram and never blocks the
program. Needs the default `jennifer` binary (`net`).

```jennifer
import "statsd.j" as statsd;

def c as statsd.Client init statsd.clientWith("127.0.0.1:8125", "web");
statsd.increment($c, "requests");        # web.requests:1|c
statsd.timing($c, "response", 42);       # web.response:42|ms
statsd.gauge($c, "queue.depth", 7);      # web.queue.depth:7|g
statsd.close($c);
```

Runnable: [`examples/modules/statsd_demo.j`](https://github.com/mplx/jennifer-lang/blob/main/examples/modules/statsd_demo.j).

## Client

```jennifer
def struct statsd.Client {
    socket as net.UDPSocket,   # the sending socket
    address as string,         # the agent "host:port"
    prefix as string           # a metric-name namespace ("" for none)
};
```

A `Client` bundles the sending socket, the agent address, and an optional
metric-name **prefix**. Value-copies share the underlying socket (the usual
handle carve-out to value semantics), so copying a `Client` is safe and cheap.
The prefix is joined to every metric name with a `.` separator, so prefix `web`
and metric `hits` send `web.hits`; an empty prefix sends the bare name.

| Call | Returns | |
| ---- | ------- | - |
| `statsd.client(host)` | `Client` | connect to `host:8125` (the default port), no prefix |
| `statsd.clientWith(address, prefix)` | `Client` | connect to a full `host:port` with a metric-name prefix |
| `statsd.close(c)` | | close the sending socket |

## Metrics

| Call | Wire line | Type |
| ---- | --------- | ---- |
| `statsd.count(c, name, value)` | `name:value\|c` | counter delta (`value` may be negative) |
| `statsd.increment(c, name)` | `name:1\|c` | counter +1 |
| `statsd.decrement(c, name)` | `name:-1\|c` | counter -1 |
| `statsd.gauge(c, name, value)` | `name:value\|g` | absolute gauge |
| `statsd.timing(c, name, ms)` | `name:ms\|ms` | timer, milliseconds |
| `statsd.set(c, name, value)` | `name:value\|s` | unique-member set (agent counts distinct values) |

All six are fire-and-forget: they format one line, send one datagram, and
return. `count` / `increment` / `decrement` adjust a counter; `gauge` sets an
absolute value; `timing` records a duration the agent aggregates into
percentiles; `set` records a distinct member (e.g. a user id) the agent counts
uniquely. Counter and gauge values are integers in this version; `set` values
are strings (any unique identifier).

## Scope

- **Fire-and-forget only.** UDP means a lost or unheard datagram is silent by
  design - there is no delivery confirmation and no error when the agent is
  down. Use it for metrics, not for data you must not lose.
- **No sample rates or tags in this version.** The StatsD `@rate` sampling
  suffix and Datadog `#tag:value` tags are not emitted; every call sends its
  metric unconditionally with no tags. Both are candidate additions.
- **Integer counter / gauge values.** Fractional gauges (e.g. a load average)
  would need a float-valued surface, not shipped here.
- **No batching.** One datagram per metric. Aggregating several metrics into a
  single packet is a possible follow-on.

## See also

- [net.md](../libraries/net.md) - the UDP surface (`listenUDP` / `sendTo`) the
  client is built on.
- [modules/index.md](index.md) - the module catalog and import rules.
