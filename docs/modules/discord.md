# `discord` - Discord Webhook client

Import with `import "discord.j" as discord;`. Post messages to a Discord channel
through a [channel Webhook](https://discord.com/developers/docs/resources/webhook),
on top of the [`http`](http.md) module - a sibling of [`gotify`](gotify.md) and
[`slack`](slack.md). Needs the default `jennifer` binary. The webhook URL is a
secret: read it from the environment or a config file, never commit it.

```jennifer
import "discord.j" as discord;

discord.send("https://discord.com/api/webhooks/1/xxx", "deploy finished");

def m as discord.Message init discord.embed(
    discord.content(discord.message(), "heads up"),
    "Deploy", "build 1234 is live", 3066993);
discord.sendMessage("https://discord.com/api/webhooks/1/xxx", $m);
```

Runnable: [`examples/modules/discord_demo.j`](https://github.com/jennifer-language/jennifer/blob/main/examples/modules/discord_demo.j).

## Plain messages

`discord.send(webhookUrl, content)` posts `{"content": content}` (the content is
Discord markdown) and returns the [`http.Response`](http.md) - Discord answers
`204 No Content` on success.

## Rich messages (embeds)

Build a message from embeds with value-semantic builders - each returns a fresh
`Message`, so they chain - then post it with `sendMessage` (or inspect the JSON
with `render`).

```jennifer
def struct discord.Message {
    content as string,        # top-level content ("" to omit; embeds must then be present)
    embeds as list of string  # pre-rendered embed JSON fragments
};
```

| Call | Returns | |
| ---- | ------- | - |
| `discord.message()` | `Message` | start an empty message |
| `discord.content(m, content)` | `Message` | set the top-level content |
| `discord.embed(m, title, description, color)` | `Message` | append an embed |
| `discord.render(m)` | `string` | render the JSON payload |
| `discord.sendMessage(webhookUrl, m)` | `http.Response` | post the built message |

`color` is the embed's left-bar colour as a **decimal** RGB integer (e.g.
`3066993` for green, `0xFF0000` = `16711680` for red). At least one of `title` /
`description` should be non-empty. All strings are JSON-escaped for you, so
quotes and newlines are safe. `content` and `embeds` are each emitted only when
present, so a plain `send`, an embeds-only message, and a content-plus-embeds
message are all valid.

## Scope

- **Channel Webhooks**, not the bot API - no bot token, gateway, slash
  commands, threads, reactions, or attachments. The channel is fixed by the
  webhook.
- **A subset of embeds** - title, description, and color. Embed fields, author,
  footer, thumbnail, and image are not built here (compose the JSON yourself and
  post via [`http`](http.md) if you need them).
- **No retry / rate-limit handling** - a non-2xx is returned as the response
  value for the caller to inspect, not thrown.

## See also

- [http.md](http.md) - the HTTP client this module builds on.
- [slack.md](slack.md) / [gotify.md](gotify.md) - the sibling notifiers.
- [modules/index.md](index.md) - the module catalog and import rules.
