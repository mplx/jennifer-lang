# `ipnet` - IP addresses and CIDR networks

Import with `import "ipnet.j" as ipnet;`. Parse and reason about **IPv4 and
IPv6** addresses and CIDR blocks: canonical formatting, membership tests, and
subnet math (netmask, broadcast). Addresses are held as raw `bytes` (4 for IPv4,
16 for IPv6); the math is bitwise. Pure Jennifer over `strings` + `convert`; runs
on **both** binaries.

```jennifer
import "ipnet.j" as ipnet;

def net as ipnet.Network init ipnet.parse("192.168.1.0/24");
def ip as ipnet.Address init ipnet.parseAddress("192.168.1.42");
def inside as bool init ipnet.contains($net, $ip);   # true
```

Runnable: [`examples/modules/ipnet_demo.j`](https://github.com/jennifer-language/jennifer/blob/main/examples/modules/ipnet_demo.j).

## Types

Both structs have public fields (read them directly), and the builder functions
are the conventional way to construct them.

```jennifer
def struct ipnet.Address { version as int, octets as bytes };   # version 4 or 6
def struct ipnet.Network { addr as Address, prefix as int };
```

- `Address.version` is `4` or `6`; `Address.octets` is the raw address bytes (4
  or 16), in network byte order.
- `Network.addr` is the base address with host bits zeroed; `Network.prefix` is
  the prefix length (`0..32` for IPv4, `0..128` for IPv6).

## Addresses

| Call | Returns | |
| ---- | ------- | - |
| `ipnet.parseAddress(s)` | `Address` | parse an IPv4 dotted-quad or IPv6 address |
| `ipnet.toString(addr)` | `string` | canonical text (RFC 5952 for IPv6) |
| `ipnet.version(addr)` | `int` | `4` or `6` |
| `ipnet.equal(a, b)` | `bool` | same version and bytes |

`parseAddress` accepts IPv6 with `::` zero-compression and a trailing embedded
IPv4 (`::ffff:192.168.1.1`). `toString` renders IPv6 canonically per RFC 5952:
lowercase, no leading zeros, and the longest run of two-or-more zero groups
compressed to `::` (leftmost on a tie). Because `equal` compares bytes, two
different spellings of the same address compare equal.

## CIDR networks

| Call | Returns | |
| ---- | ------- | - |
| `ipnet.parse(cidr)` | `Network` | parse `address/prefix` (host bits zeroed) |
| `ipnet.networkString(net)` | `string` | render as `address/prefix` |
| `ipnet.contains(net, addr)` | `bool` | is the address in the network? |
| `ipnet.netmask(net)` | `Address` | the netmask (e.g. `255.255.255.0`) |
| `ipnet.broadcast(net)` | `Address` | the last address (IPv4 broadcast) |

`parse` zeroes the host bits, so `ipnet.parse("192.168.1.42/24")` has base
`192.168.1.0`. `contains` returns `false` for a version mismatch (an IPv4
address is never inside an IPv6 network). `broadcast` sets every host bit: for
IPv4 that is the broadcast address, for IPv6 the last address in the block.

```jennifer
def allowed as list of ipnet.Network init [ipnet.parse("10.0.0.0/8"), ipnet.parse("192.168.0.0/16")];
def client as ipnet.Address init ipnet.parseAddress("10.4.5.6");
def ok as bool init false;
for (def net in $allowed) {
    if (ipnet.contains($net, $client)) { $ok = true; }
}
```

## Errors

Malformed input throws `Error{kind: "ipnet"}` (a bad octet, too few / many
groups, multiple `::`, a bad hex digit, a missing `/prefix`, or an out-of-range
prefix) - catch it with `try` / `catch`.

## Scope

- **Address and prefix math, not a resolver.** No DNS, no interface
  enumeration; hostname / interface lookups live in the `net` library.
- **Lenient decimal octets.** `parseAddress` reads IPv4 octets as plain decimal,
  so leading zeros are accepted as decimal (`192.168.001.001` = `192.168.1.1`),
  not rejected or read as octal.
- **IPv4-mapped IPv6 renders as hex.** `::ffff:192.168.1.1` round-trips at the
  byte level but `toString` renders it in pure hex form (`::ffff:c0a8:101`), not
  the dotted-quad convenience form.
- **No subnet-of-subnet test in v1.** `contains` tests an address in a network;
  network-in-network containment is not modelled yet.

## See also

- [net.md](../libraries/net.md) - sockets, TLS, and DNS lookups.
- [strings.md](../libraries/strings.md) - the text surface the parser builds on.
- [modules/index.md](index.md) - the module catalog and import rules.
