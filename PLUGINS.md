# Proxima Plugin Development Guide

Proxima plugins are JavaScript files placed in the `plugins/` folder of your
Proxima data directory. Each plugin runs in its own sandboxed runtime; a plugin
that errors or hangs is isolated and never crashes the proxy.

## Plugin directory by platform

| OS      | Plugin directory                                   |
| ------- | -------------------------------------------------- |
| macOS   | `~/Library/Application Support/proxima/plugins/`   |
| Linux   | `$XDG_CONFIG_HOME/proxima/plugins/` (or `~/.config/proxima/plugins/`) |
| Windows | `%APPDATA%\proxima\plugins\`                        |

Use the **Open plugins folder** button on the Plugins page to open this folder
in your file manager, then drop `.js` files in and press **Reload**.

## Plugin structure

A plugin exports a `meta` object and one or more hook functions:

```javascript
export const meta = {
  name: "My Plugin",
  version: "1.0.0",
  description: "What it does",
  author: "Your Name"   // your own name or handle
};

export function onRequest(ctx) {
  // mutate ctx and return it
  ctx.request.headers['X-Example'] = '1';
  return ctx;
}
```

## Hooks

| Hook            | When it runs                       | Useful fields                              |
| --------------- | ---------------------------------- | ------------------------------------------ |
| `onRequest`     | before a request is forwarded      | `ctx.request.{method,url,headers,body}`    |
| `onResponse`    | after a response is received       | `ctx.response.{statusCode,headers,body}`   |
| `onIntercept`   | before a request enters the queue  | set `ctx.action = "forward"` or `"drop"`   |
| `onFuzzResult`  | after each fuzzer result           | `ctx.result.{statusCode,responseSize,...}` |

Each hook receives a context object and should return it (possibly modified).

## The `proxima` API

| Call                          | Description                                          |
| ----------------------------- | ---------------------------------------------------- |
| `proxima.log(msg)`            | Write a message to the Proxima log                   |
| `proxima.alert(msg)`          | Raise a plugin notification                          |
| `proxima.http.send(req)`      | Make an outbound HTTP request (loopback is blocked)  |
| `proxima.store.get(key)`      | Read a value persisted for this plugin's session     |
| `proxima.store.set(key, val)` | Persist a value for this plugin's session            |

`proxima.http.send` rejects requests to `localhost`, `127.0.0.1`, and `::1` to
prevent plugins from reaching the local admin API (anti-SSRF).

## Examples

See [`examples/plugins/`](./examples/plugins/) for `header-injector.js`,
`5xx-logger.js`, and `drop-binary.js`.

## Author field

Set the `author` field in `meta` to your own name or handle.
