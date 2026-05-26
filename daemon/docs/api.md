# pm2-go daemon — web API

The daemon exposes two surfaces:

- **Unix-socket IPC** at `~/.pm2-go/rpc.sock` — used by the CLI. No auth (file-mode 0600).
- **HTTP web API** on a TCP port — used by the Next.js UI and any external integration. Bearer-token auth. **This document covers the HTTP web API.**

The web API is **disabled** by default. Enable it once with `pm2-go web --port 9615`, which writes `~/.pm2-go/api-port` and `~/.pm2-go/api-token`, then restart the daemon (`pm2-go kill && pm2-go ping`). Inside docker-compose, set `PM2_GO_API_ADDR` and `PM2_GO_API_TOKEN` env vars instead.

---

## Base URL & auth

```
GET  /healthz                  no auth
*    /v1/*                     Bearer token required
```

Header:

```
Authorization: Bearer <token>
```

Token source on the daemon (first match wins):

1. `PM2_GO_API_TOKEN` env var
2. `~/.pm2-go/api-token` file

Token on the client side: `pm2-go web --token` prints the current one, `pm2-go web --regenerate-token` rotates.

`401 Unauthorized` with no body is returned on missing or wrong token.

## CORS

`Access-Control-Allow-Origin: *` by default. Override with `PM2_GO_API_CORS=<origin>`. Pre-flight `OPTIONS` returns `204`.

## Content types

- Requests: `application/json` when there's a body. Otherwise empty.
- Responses: `application/json` for most endpoints; `application/x-ndjson` for `/v1/logs/stream` (one JSON object per line, server flushes on every line).

## Error model

```json
{ "error": "<message>" }
```

Status codes used:

| Status | Meaning |
| --- | --- |
| `200` | OK |
| `400` | Malformed request (bad JSON, missing query param) |
| `401` | Missing / wrong bearer token |
| `404` | Named app does not exist on the daemon |
| `502` | Internal error (propagated as JSON `{"error": "..."}`) |

---

## Endpoints

In the curl examples below:

```bash
URL=http://127.0.0.1:9615
TOKEN=$(pm2-go web --token)
HDR="Authorization: Bearer $TOKEN"
```

### `GET /healthz`

Liveness probe. No auth.

```bash
curl -s $URL/healthz
# {"ok":true}
```

### `GET /v1/procs`

List every runtime instance. One row per cluster instance (so a 3-instance app yields 3 rows).

```bash
curl -sH "$HDR" $URL/v1/procs | jq
```

Response:

```json
{
  "procs": [
    {
      "id": "api-0",
      "app_id": "api",
      "name": "api",
      "instance_id": 0,
      "namespace": "prod",
      "state": "online",
      "pid": 18234,
      "started_at": "2026-05-26T12:00:00+08:00",
      "uptime_seconds": 134,
      "restarts": 2,
      "unstable_restarts": 0,
      "cpu": 4.7,
      "mem": 38912000,
      "exit_code": 0
    }
  ]
}
```

Possible `state` values: `launching`, `online`, `stopping`, `stopped`, `errored`, `waiting_restart`, `online_restarting`.

### `GET /v1/specs`

The persisted spec list (what gets written to `~/.pm2-go/dump.json`). One row per app, not per runtime.

```bash
curl -sH "$HDR" $URL/v1/specs | jq
```

```json
{
  "specs": [
    {
      "id": "api",
      "name": "api",
      "script": "/srv/api/index.js",
      "interpreter": "node",
      "args": ["--port", "3000"],
      "cwd": "/srv/api",
      "env": { "NODE_ENV": "production" },
      "instances": 4,
      "exec_mode": "fork",
      "namespace": "prod",
      "autorestart_disabled": false,
      "max_restarts": 16,
      "min_uptime": 1000000000,
      "kill_timeout": 1600000000,
      "watch": ["/srv/api/src"],
      "ignore_watch": ["*.log"]
    }
  ]
}
```

Durations (`min_uptime`, `kill_timeout`, `restart_delay`) are nanoseconds — Go's `time.Duration` JSON encoding. `max_memory_restart` is bytes.

### `GET /v1/describe?name=<name>`

Spec + every runtime instance for one app.

```bash
curl -sH "$HDR" "$URL/v1/describe?name=api" | jq
```

```json
{
  "spec":  { /* same shape as /v1/specs item */ },
  "procs": [ /* same shape as /v1/procs items */ ]
}
```

`404` if the name doesn't exist.

### `POST /v1/start`

Add or replace an app. Two modes:

**a. Create from a spec body** — full app configuration as JSON:

```bash
curl -sH "$HDR" -X POST $URL/v1/start \
  -H "Content-Type: application/json" \
  -d '{
    "name": "api",
    "script": "/srv/api/index.js",
    "interpreter": "node",
    "instances": 4,
    "namespace": "prod",
    "env": { "NODE_ENV": "production" },
    "watch": ["/srv/api/src"],
    "max_memory_restart": 314572800
  }'
# {"name":"api"}
```

If an app named `api` already exists, it's stopped and re-spawned with the new spec. Logs on disk are preserved.

Minimal spec — only `name` and `script` are required. Defaults applied: `instances=1`, `exec_mode=fork`, `namespace=default`, `autorestart=on`, `kill_timeout=1600ms`, `min_uptime=1s`, `max_restarts=16`.

**b. Start an existing (stopped) app by name** — no body:

```bash
curl -sH "$HDR" -X POST "$URL/v1/start?name=api"
# {"ok":true}
```

### `POST /v1/stop?name=<name>`

Send SIGTERM to every instance, then SIGKILL after `kill_timeout`. The spec stays — the app is in `stopped` state and can be started again with `POST /v1/start?name=…`.

```bash
curl -sH "$HDR" -X POST "$URL/v1/stop?name=api"
# {"ok":true}
```

### `POST /v1/restart?name=<name>`

Stop then start. The runtime's restart counter is incremented by one (so it reflects user-initiated restarts on the dashboard).

```bash
curl -sH "$HDR" -X POST "$URL/v1/restart?name=api"
# {"ok":true}
```

### `POST /v1/reload?name=<name>`

Graceful reload — sends SIGUSR2 to every instance. Apps that don't handle SIGUSR2 will terminate and be restarted by the supervisor (autorestart path), so behaviour matches a hard restart for non-cooperating apps.

```bash
curl -sH "$HDR" -X POST "$URL/v1/reload?name=api"
```

### `POST /v1/delete?name=<name>`

Stops every instance and removes the spec from the dump. **Log files on disk are NOT deleted** (so post-mortem inspection works).

```bash
curl -sH "$HDR" -X POST "$URL/v1/delete?name=api"
```

### `POST /v1/save`

Snapshot the current spec list to `~/.pm2-go/dump.json` (atomic write via tmp + rename). The dump is also auto-saved on every mutation, so this is mostly a debugging knob.

```bash
curl -sH "$HDR" -X POST $URL/v1/save
```

### Bulk endpoints

```bash
curl -sH "$HDR" -X POST $URL/v1/start-all     # start every persisted spec
curl -sH "$HDR" -X POST $URL/v1/stop-all      # stop every running app
curl -sH "$HDR" -X POST $URL/v1/delete-all    # stop + remove every spec
```

All return `{"ok": true}` synchronously after the work is dispatched.

### `GET /v1/logs?name=<name>&n=<lines>&stream=<out|err|>`

Tail the most recent `n` lines from each runtime's stdout / stderr.

- `n` — defaults to 50 if omitted.
- `stream` — `out` or `err` to filter; omit for both.

```bash
curl -sH "$HDR" "$URL/v1/logs?name=api&n=20" | jq
```

```json
{
  "streams": {
    "api-0:out": ["listening on :3000", "got request /healthz"],
    "api-0:err": []
  }
}
```

Empty streams are always `[]` (never `null`).

### `GET /v1/logs/stream?name=<name>`

Live ndjson stream of stdout + stderr from every runtime. The server keeps the connection open and writes one JSON object per log line. Disconnect to stop.

```bash
curl -NsH "$HDR" "$URL/v1/logs/stream?name=api"
# {"runtime":"api-0","stream":"out","time":1779781146174,"text":"listening on :3000\n"}
# {"runtime":"api-0","stream":"err","time":1779781146210,"text":"warning: fd leak\n"}
# ...
```

- `runtime` — runtime id (`<name>` for single-instance, `<name>-<i>` for cluster).
- `stream` — `out` or `err`.
- `time` — unix milliseconds.
- `text` — usually ends in `\n`. Don't rely on it.

Slow clients drop messages: the server uses non-blocking sends into per-subscriber buffered channels (cap 64).

### `GET /v1/metrics?name=<name>`

Rolling CPU% / RSS samples per runtime, ~1Hz, last 5 minutes (300 samples).

```bash
curl -sH "$HDR" "$URL/v1/metrics?name=api" | jq
```

```json
{
  "series": {
    "api-0": [
      { "time": 1779781140000, "cpu": 4.2, "mem": 39845888 },
      { "time": 1779781141000, "cpu": 5.1, "mem": 39845888 }
    ],
    "api-1": [ ... ]
  }
}
```

CPU is percent of one core. Mem is bytes (RSS from `/proc/<pid>/status:VmRSS`).

---

## Spec object reference

Every field on the `Spec` JSON shape:

| Field | Type | Notes |
| --- | --- | --- |
| `id` | string | Stable identifier, derived from `name` if omitted. |
| `name` | string | **Required**. Display name. Primary key. |
| `script` | string | **Required**. Absolute path or executable name on PATH. |
| `args` | string[] | Args passed to `script`. |
| `interpreter` | string | e.g. `node`, `python3`, `/bin/bash`. Empty = `script` runs directly. |
| `interpreter_args` | string[] | Args passed to the interpreter, before `script`. |
| `cwd` | string | Working directory. |
| `env` | map[string]string | Environment overrides. Wins over `env_files`. |
| `env_files` | string[] | Paths to `.env` files (KEY=VALUE per line, `#` comments). |
| `instances` | int | Default 1. >1 = cluster mode (each instance gets `PM2_INSTANCE_ID`, `NODE_APP_INSTANCE`). |
| `exec_mode` | "fork" \| "cluster" | Currently only `fork` is implemented — `cluster` is reserved. |
| `namespace` | string | Filter label. Default `default`. |
| `autorestart_disabled` | bool | If true, don't restart on exit. |
| `max_restarts` | int | Cap on unstable restarts before marking `errored`. 0 = unlimited (default 16). |
| `min_uptime` | int (ns) | Exits earlier than this count as unstable. Default 1s. |
| `restart_delay` | int (ns) | Fixed sleep between attempts. |
| `exp_backoff_restart` | bool | Exponentially back off on unstable restarts (capped at 64s). |
| `kill_timeout` | int (ns) | SIGTERM → SIGKILL grace. Default 1.6s. |
| `max_memory_restart` | uint64 | Restart if RSS exceeds this (bytes). |
| `stop_exit_codes` | int[] | Don't restart if the process exited with one of these codes. |
| `watch` | string[] | Paths to watch with fsnotify (debounced restart on change). |
| `ignore_watch` | string[] | Globs matched against the changed file's basename. `node_modules` and dotfiles always skipped. |
| `out_path`, `err_path` | string | Override log paths (defaults `~/.pm2-go/logs/<runtime>-out.log`). |
| `merge_logs` | bool | Reserved. |
| `log_date_format` | string | Reserved. |
| `log_max_size_mb` | int | Enable lumberjack rotation at this size. 0 = no rotation. |
| `log_max_backups` | int | Lumberjack rotation. |
| `log_max_age_days` | int | Lumberjack rotation. |
| `log_compress` | bool | Lumberjack gzip rotated files. |
| `cron_restart` | string | Reserved (parsed, not enforced yet). |
| `created_at`, `updated_at` | RFC3339 strings | Set by the daemon. |

### Running a raw command line

To submit a multi-line shell script via the API:

```bash
curl -sH "$HDR" -X POST $URL/v1/start \
  -H "Content-Type: application/json" \
  -d @- <<'JSON'
{
  "name": "tail-bridge",
  "script": "/bin/sh",
  "args": ["-c", "cd /srv/app && exec node index.js | tee -a /var/log/app.log"]
}
JSON
```

Same effect as `pm2-go start --shell "…"` or the UI's "Run in shell" toggle.

---

## IPC (Unix socket) — for reference

The CLI talks to the daemon over HTTP/1.1 carried on a Unix socket at `~/.pm2-go/rpc.sock`. Paths are namespaced under `/v1/` and mirror the web API, with two extras:

| Path | Method | Notes |
| --- | --- | --- |
| `/v1/ping` | `GET` | Returns `{"pid":..., "started":..., "ok":true}` |
| `/v1/start-spec` | `POST` | Same as web API `/v1/start` with JSON body. Kept separate from `/v1/start` (which takes `{name}`) for clarity. |
| `/v1/shutdown` | `POST` | Tell the daemon to shut down. The CLI's `pm2-go kill` calls this. |

Auth: socket file mode `0600`, owned by the daemon user. No bearer token.

---

## Versioning & stability

All endpoints live under `/v1/*`. Breaking changes will bump to `/v2/*`; new fields on existing responses can show up at any time. Clients should ignore unknown JSON keys.

The IPC protocol (Unix socket) is considered internal to the CLI ↔ daemon pair and is not stability-guaranteed across daemon versions. The web API is the integration contract.
