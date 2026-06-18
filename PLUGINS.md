# Proxima Plugin Development Guide

Proxima supports JavaScript plugins that hook into the proxy pipeline. Each
plugin runs in its own sandboxed runtime; a plugin that throws or hangs is
isolated and never crashes the proxy.

---

## Plugin directory by platform

| OS      | Plugin directory |
|---------|------------------|
| macOS   | `~/Library/Application Support/proxima/plugins/` |
| Linux   | `~/.config/proxima/plugins/` (or `$XDG_CONFIG_HOME/proxima/plugins/`) |
| Windows | `%APPDATA%\proxima\plugins\` |

Drop any `.js` file into your platform's plugin directory. Plugins are loaded on
startup. To reload without restarting, open the **Plugins** page in the admin UI
and click **Reload** (or use the **Open plugins folder** button to jump straight
to the directory).

---

## Plugin structure

A plugin exports a `meta` object and one or more hook functions:

```javascript
export const meta = {
  name: "My Plugin",
  version: "1.0.0",
  description: "A short description of what this plugin does",
  author: "Your Name"   // your own name or handle
};

// Hook: called before each request is forwarded upstream.
export function onRequest(ctx) {
  ctx.request.headers['X-My-Header'] = 'value';
  return ctx;
}

// Hook: called after each response is received from upstream.
export function onResponse(ctx) {
  if (ctx.response.statusCode === 403) {
    proxima.log('Got 403 on ' + ctx.request.url);
  }
  return ctx;
}

// Hook: called when a request enters the intercept queue.
// Set ctx.action = 'forward' or 'drop' to auto-handle it.
export function onIntercept(ctx) {
  return ctx;
}

// Hook: called for each fuzzer result.
export function onFuzzResult(ctx) {
  if (ctx.result.statusCode === 500) {
    proxima.alert('Possible error on payload: ' + JSON.stringify(ctx.result.payloadValues));
  }
  return ctx;
}
```

All hooks are optional — export only the ones you need. **Every hook must return
its context object** (modified or not).

---

## Hook reference

### `onRequest(ctx)`

| Field | Type | Description |
|---|---|---|
| `ctx.request.method` | string | HTTP method |
| `ctx.request.url` | string | Full URL |
| `ctx.request.headers` | object | Request headers (key → value) |
| `ctx.request.body` | string | Request body as UTF-8 |

Modify `ctx.request` fields directly, then return `ctx`.

### `onResponse(ctx)`

Includes the same `ctx.request` as above (read-only here), plus:

| Field | Type | Description |
|---|---|---|
| `ctx.response.statusCode` | number | HTTP status code |
| `ctx.response.headers` | object | Response headers |
| `ctx.response.body` | string | Response body as UTF-8 |

### `onIntercept(ctx)`

Same `ctx.request` as `onRequest`. Additionally:

| `ctx.action` | Effect |
|---|---|
| `""` (default) | Show in the intercept queue for manual review |
| `"forward"` | Auto-forward without showing in the queue |
| `"drop"` | Auto-drop without showing in the queue |

### `onFuzzResult(ctx)`

| Field | Type | Description |
|---|---|---|
| `ctx.result.attackId` | string | Parent attack ID |
| `ctx.result.requestIndex` | number | Sequence number |
| `ctx.result.payloadValues` | object | Position → value map |
| `ctx.result.statusCode` | number | Response status |
| `ctx.result.responseSize` | number | Response body size in bytes |
| `ctx.result.responseTimeMs` | number | Round-trip time in milliseconds |
| `ctx.result.isError` | boolean | Whether the request failed |
| `ctx.result.tags` | string[] | Add your own tags here |

---

## The `proxima` global

### `proxima.log(message)`
Write a structured log line tagged with your plugin name.

### `proxima.alert(message)`
Raise a notification visible on the Proxima Plugins page.

### `proxima.http.send(request)`
Make an outbound HTTP request.

```javascript
const res = proxima.http.send({
  method: 'POST',
  url: 'https://hooks.example.com/notify',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ url: ctx.request.url })
});
proxima.log('Notification sent: ' + res.statusCode);
```

**Security:** `proxima.http.send` rejects requests to `localhost`, `127.0.0.1`,
and `::1` to prevent plugins from reaching the local admin API (anti-SSRF).

### `proxima.store.get(key)` / `proxima.store.set(key, value)`
A per-plugin, in-memory key-value store. Values persist across requests within a
session, but not across restarts.

---

## Timeouts and error handling

- Each hook invocation has a **5-second timeout**. Long-running hooks are cancelled.
- If a plugin throws or times out, the error is recorded (visible on the Plugins
  page as **Last Error**) and that plugin is skipped for the current request.
- Plugins **never** crash the proxy.

---

## Example plugins

See [`examples/plugins/`](./examples/plugins/) for ready-to-use examples:

| File | What it does |
|---|---|
| `header-injector.js` | Adds an `X-Proxima-Plugin` header to every proxied request |
| `5xx-logger.js` | Logs all 5xx responses to the Proxima log |
| `drop-binary.js` | Auto-drops image/video/audio requests from the intercept queue |

---

## Sharing plugins

Post plugins to GitHub Gists, the Proxima GitHub Discussions, or security
community channels. Set the `author` field in `meta` to your own name or handle.
