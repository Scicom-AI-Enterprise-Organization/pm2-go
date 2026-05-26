# pm2-go

A [PM2](https://github.com/Unitech/pm2)-style process manager written in Go, paired
with an enterprise Next.js admin UI for managing the daemon over a web browser.

- **`daemon/`** — single static Go binary (`pm2-go`) with the daemon, CLI, IPC,
  and embedded web API server.
- **`src/`** — Next.js 16 enterprise template (Auth.js v5 + Prisma + RBAC) with
  a `/processes` UI that talks to the daemon's web API.

The two pieces are independent: you can run only the binary if you don't want a
web UI, or run the UI against a daemon on another host.

## Highlights

- **Single static binary** — no node, no system deps; ~6 MB on Linux/amd64.
- **PM2-compatible verbs**: `start`, `stop`, `restart`, `reload`, `delete`,
  `list`, `show`, `logs`, `save`, `resurrect`, `dump`, `monit`, `startup`,
  `ping`, `kill`, `web`.
- **Quoted commands & shell mode** — `pm2-go start "node app.js --port 3000 --env prod"`
  auto-splits into script+args. Use `--shell "cd /srv && node app.js | tee log"`
  to wrap the command in `/bin/sh -c "…"` for pipes, redirects, and expansion.
  The UI exposes the same via a "Run in shell" toggle on the create/edit form.
- **Cluster mode** — start N instances with `-i N`; each gets
  `PM2_INSTANCE_ID` / `NODE_APP_INSTANCE` in its env.
- **Ecosystem files** — `pm2-go start ecosystem.config.json` (JSON; YAML
  scheduled). Supports `name`, `script`, `args`, `interpreter`, `cwd`, `env`,
  `instances`, `namespace`, `autorestart`, `max_restarts`, `min_uptime`,
  `restart_delay`, `exp_backoff_restart`, `kill_timeout`,
  `max_memory_restart` ("300M", "1G", etc.), `stop_exit_codes`, `watch`,
  `ignore_watch`, log rotation knobs (`log_max_size_mb`, …), `cron_restart`.
- **Watch mode** — fsnotify-backed; debounced restart on change. `node_modules`
  and dotfiles auto-skipped.
- **Graceful reload** — `reload` sends `SIGUSR2`; falls back to restart for apps
  that don't handle it.
- **Auto-restart with backoff** — `min_uptime` detects flaps;
  `exp_backoff_restart` doubles the wait; `max_restarts` stops at the cap.
- **Memory-triggered restart** — `max_memory_restart` reboots processes that
  exceed an RSS threshold.
- **Log rotation** — per-process; size/age/backups via lumberjack.
- **Persistence** — `~/.pm2-go/dump.json` is the canonical state; restored on
  daemon boot. `pm2-go save` to snapshot manually, `pm2-go startup` to wire up
  a systemd user unit.
- **Auth'd web API** — bearer-token gated. The Next.js UI consumes it via a
  server-side proxy so the token never reaches the browser.
- **RBAC-aware web UI** — new permissions (`processes:{read,write,delete,logs}`)
  and a pre-seeded `pm2-operator` role grant the UI without making everyone an
  admin.

## How to run

### Prerequisites

- **Go 1.23+** for building the daemon (`go.dev/dl` or `~/go-toolchain/go/bin/go` if you used this repo's installer).
- **Node 20+** and **npm** for the web UI.
- **Docker** (or a local Postgres 14+) for the Auth.js session/RBAC database.
- Linux (the daemon uses `/proc` for metrics and POSIX signals).

### 1 — Build & try the daemon (no DB needed)

```bash
cd daemon
make build                         # → bin/pm2-go
make install                       # → ~/.local/bin/pm2-go (override with PREFIX=)

# Start something. The daemon auto-spawns on the first command.
pm2-go start /path/to/server.js --name api -i 4 \
    --watch src --ignore-watch '*.log' --max-memory-restart 300M

# Or pass a long command line as one quoted string — it auto-splits into
# script + args, the same way a shell would:
pm2-go start "node /srv/app/index.js --port 3000 --env production" --name api

# Or use --shell for pipes, redirects, and shell expansion:
pm2-go start --shell "cd /srv/app && exec node index.js | tee -a /var/log/app.log" --name api

pm2-go list                        # status table
pm2-go logs api                    # tail + follow (Ctrl-C to stop)
pm2-go monit                       # full-screen TTY dashboard
pm2-go restart api
pm2-go save                        # snapshot to ~/.pm2-go/dump.json
pm2-go startup --install           # write & enable a systemd --user unit
pm2-go kill                        # shut the daemon down
```

State lives in `~/.pm2-go/` (override with `PM2_GO_HOME`). Add `~/.local/bin` to your `PATH` (or run `daemon/bin/pm2-go` directly) and skip `make install` if you prefer to keep the binary in-tree.

### 2 — Turn on the web API

The web API is **off by default**. Configure it once:

```bash
pm2-go web --port 9615             # generates ~/.pm2-go/api-token + saves port
pm2-go web --token                 # print the current token
pm2-go web --regenerate-token      # rotate it

# Daemon must be restarted to bind the port:
pm2-go kill
pm2-go ping                        # auto-spawns, daemon now serves :9615
curl -H "Authorization: Bearer $(pm2-go web --token)" \
     http://127.0.0.1:9615/v1/procs | jq
```

### 3 — Bring up the Next.js admin UI

From the repo root (parent of `daemon/`):

```bash
# Configure
cp .env.example .env               # then edit AUTH_SECRET (openssl rand -base64 32)
                                   # and DATABASE_URL if you're not using the docker postgres

# Database
docker compose up -d db            # starts the bundled Postgres on :5432
npm install
npm run db:migrate                 # asks for a migration name on first run; type anything
npm run db:seed                    # admin user + roles (admin, member, pm2-operator)

# Dev server
npm run dev
```

Open <http://localhost:3000>, sign in with `admin@example.com` / `admin1234` (change before deploying), then click **Processes** in the sidebar. From there you can:

- **List / search / filter** processes by name or namespace, with a live 2-second auto-refresh.
- **Create** a new process via `/processes/new` — every CLI flag is exposed in the form.
- **Import** an `ecosystem.config.json` payload via `/processes/import`.
- **Inspect** a process at `/processes/<name>` — runtime instances, CPU/mem sparklines, env, watch paths.
- **Edit** an existing spec at `/processes/<name>/edit` (saves are add-or-replace on the daemon).
- **Stream live logs** at `/processes/<name>/logs` with pause / clear / colour-coded stderr.
- **Daemon-wide toolbar** — Save dump, Start all, Stop all, Delete all.
- **Per-row actions** — restart, reload (SIGUSR2), stop, delete.

The token never reaches the browser — the Next.js server proxies all daemon calls.

### 4 — Production deployment

```bash
# Daemon as a systemd --user service:
pm2-go startup --install
systemctl --user daemon-reload
systemctl --user enable --now pm2-go

# Next.js app:
npm run build && npm start
# or:
docker compose up --build -d
docker compose exec app npx prisma migrate deploy
docker compose exec app npx tsx prisma/seed.ts
```

If the UI and daemon are on different hosts, point the UI at the remote daemon:

```env
PM2_GO_DAEMON_URL=http://daemon-host.internal:9615
PM2_GO_DAEMON_TOKEN=<paste ~/.pm2-go/api-token from that host>
```

### Running the test suite

The Go daemon has unit + integration tests covering the supervisor, IPC, web
API, ecosystem parsing, log capture, watcher, metrics, and CLI helpers. Tests
run against a temp `PM2_GO_HOME` (per `t.TempDir()`) so they never touch
`~/.pm2-go`.

```bash
cd daemon
make test            # go test ./...
make test-race       # go test -race ./...   (catches concurrency bugs)
make test-cover      # go test -cover ./...  (per-package coverage)
make lint            # go vet ./...
```

Direct `go test` examples (useful while iterating):

```bash
# One package, verbose:
~/go-toolchain/go/bin/go test -v ./internal/daemon

# A single test by name:
~/go-toolchain/go/bin/go test -run TestSupervisorClusterMode ./internal/daemon

# All tests, race detector, no cache, with coverage profile:
~/go-toolchain/go/bin/go test -race -count=1 -coverprofile=cover.out ./...
~/go-toolchain/go/bin/go tool cover -html=cover.out
```

The integration tests spawn real `/bin/sleep`, `/bin/true`, `/bin/false`, and
`/bin/sh` processes, so they need a POSIX userland (any Linux). A full run
takes ~5 seconds with `-race`.

If a test fails intermittently, it's almost always a timing race — bump the
`waitUntil` deadlines in `daemon/internal/daemon/supervisor_test.go` (default
3s) or rerun with `-count=1` to defeat the cache.

### Troubleshooting

| Symptom | Fix |
| --- | --- |
| `daemon did not become ready within 5s` | Check `~/.pm2-go/daemon.log`. Common cause: ecosystem file lookup error. |
| UI says "Daemon unreachable" | Confirm `pm2-go web` was configured and you've `pm2-go kill && pm2-go ping` to rebind. Check `PM2_GO_DAEMON_URL` / `PM2_GO_DAEMON_TOKEN` in `.env`. |
| Login page 500s | DB not seeded; run `npm run db:migrate && npm run db:seed`. |
| `/processes` not in the sidebar | Your account doesn't have `processes:read`. As admin, go to `/admin/users` and assign the `pm2-operator` (or `admin`) role. |
| Watch mode doesn't fire | The watcher skips `node_modules` and dotfiles. Use `--watch <dir>` with a parent directory of the changed file. |

## Cross-host deploys

The Next.js app talks to the daemon over HTTP. Set in `.env`:

```env
PM2_GO_DAEMON_URL=http://daemon-host.internal:9615
PM2_GO_DAEMON_TOKEN=<paste the api-token from that host>
```

On a single dev box, omit `PM2_GO_DAEMON_TOKEN` — the lib falls back to
reading `~/.pm2-go/api-token` directly.

## Architecture

```
┌──────────────┐    HTTP+ndjson    ┌──────────────┐    Unix socket    ┌──────────────┐
│  Browser     │ ◀──────────────▶ │  Next.js app │ ◀──────────────▶ │  pm2-go CLI  │
└──────────────┘                  │  (RBAC, UI)  │                  └──────┬───────┘
                                  └──────┬───────┘                         │
                                         │ HTTP + bearer token             │ unix
                                         ▼                                 ▼
                            ┌────────────────────────────────────────────────────┐
                            │  pm2-go daemon (single Go binary, runs detached)   │
                            │  ├ Unix socket IPC (CLI)                           │
                            │  ├ Authenticated TCP web API (UI)                  │
                            │  ├ Supervisor: spawn / monitor / restart / metrics │
                            │  ├ Per-process log capture + rotation              │
                            │  ├ Watch mode (fsnotify)                           │
                            │  └ dump.json persistence                           │
                            └────────────────────────────────────────────────────┘
```

The CLI auto-spawns the daemon on the first command via `pm2-go __daemon`. The
web API is only bound if `pm2-go web` has been configured at least once (writes
`~/.pm2-go/api-port` and `~/.pm2-go/api-token`).

## RBAC for the UI

Seeded by `npm run db:seed`:

| Permission         | Description                              |
| ------------------ | ---------------------------------------- |
| `processes:read`   | List + view processes                    |
| `processes:write`  | Start, restart, reload, stop processes   |
| `processes:delete` | Remove processes from the dump           |
| `processes:logs`   | Tail and stream logs                     |

Roles:

- `admin` — every permission, including all process perms.
- `pm2-operator` — every process perm but no user/role admin access.
- `member` — none.

## What's not yet implemented

- YAML ecosystem files (JSON only for now).
- `pm2 deploy` / SSH-based deployment workflow.
- Module system (`pm2 install <module>`).
- `cron_restart` (parsed but not enforced).
- Multi-host clustering (one daemon per host).
- Windows (relies on POSIX signals / `/proc`).

## Original enterprise template

This repo started as a Next.js 16 enterprise starter (Auth.js v5, Prisma,
RBAC, admin UI). The original docs for that template are kept verbatim below
since most of it still applies to the UI side of the project.

---

A Next.js 16 starter for building internal/enterprise apps. Comes wired with:

- **Auth.js v5** — credentials (email + password), Microsoft Entra ID (Azure AD), Google, Keycloak, SAML
- **RBAC** — roles + permissions, enforced by Next.js proxy middleware and server helpers
- **Prisma + Postgres** — users, accounts, sessions, roles, permissions, invitations
- **Authed app shell** — collapsible sidebar nav, top bar with theme toggle and user menu
- **Profile** — `/profile` lets users update their name and set/change their password (works for both credentials and SSO users)
- **Admin UI** — `/admin/users` (assign roles), `/admin/roles` (toggle permissions), `/admin/organization` (shareable invite links)
- **Processes** — `/processes` lists pm2-go managed apps, with start/stop/restart/reload/delete and live log streaming
- **AI Workspace** — `/ai` interactive demo combining every showcase AI pattern (chatbot, structured flow, agent assist, voice, summary cards)
- **Showcase** — the original component gallery preserved at `/showcase/*`
- React 19, Tailwind CSS v4, Radix UI

## Default credentials

After running `npm run db:seed`, sign in at `/login` with:

| Email | Password |
| --- | --- |
| `admin@example.com` | `admin1234` |

Override before seeding by setting `SEED_ADMIN_EMAIL` and `SEED_ADMIN_PASSWORD` in `.env`. **Change these before deploying anywhere reachable.**

## Auth providers

Each SSO provider is enabled only if its env vars are set. See `.env.example`.

- **Azure AD / Microsoft Entra ID** — `AUTH_AZURE_AD_CLIENT_ID`, `AUTH_AZURE_AD_CLIENT_SECRET`, `AUTH_AZURE_AD_TENANT_ID`
- **Google** — `AUTH_GOOGLE_CLIENT_ID`, `AUTH_GOOGLE_CLIENT_SECRET`
- **Keycloak** — `AUTH_KEYCLOAK_CLIENT_ID`, `AUTH_KEYCLOAK_CLIENT_SECRET`, `AUTH_KEYCLOAK_ISSUER`
- **SAML** — `AUTH_SAML_ENTRY_POINT`, `AUTH_SAML_ISSUER`, `AUTH_SAML_IDP_CERT`. Implemented with `@node-saml/node-saml`. SP routes are at `/api/auth/saml/login` and `/api/auth/saml/callback`. For multi-tenant SAML, point a generic OIDC provider at [BoxyHQ Jackson](https://boxyhq.com/docs/jackson/overview) instead.

OAuth/OIDC redirect URI: `${AUTH_URL}/api/auth/callback/<provider>`. SAML ACS: `${AUTH_URL}/api/auth/saml/callback`.

## Tech stack

- [Next.js 16](https://nextjs.org)
- [React 19](https://react.dev)
- [Auth.js v5](https://authjs.dev)
- [Prisma 6](https://www.prisma.io) + Postgres
- [Tailwind CSS v4](https://tailwindcss.com)
- [Radix UI](https://www.radix-ui.com)
- [Go 1.23+](https://go.dev) (for the daemon)
- [fsnotify](https://github.com/fsnotify/fsnotify), [lumberjack](https://github.com/natefinch/lumberjack) (Go deps)
