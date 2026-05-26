# pm2-go daemon

A PM2-style process manager as a single Go binary.

```
$ pm2-go help
pm2-go — process manager

USAGE
  pm2-go <command> [args]

COMMANDS
  delete (del, rm)        Stop and remove a process by name (or 'all')
  dump                    Print the saved specs as JSON
  help                    Show help for a command
  kill                    Shut down the daemon (stops all managed processes)
  list (ls, ps, status)   List all managed processes
  logs                    Tail logs for one or all processes
  monit                   Live TTY dashboard of all processes
  ping                    Check whether the daemon is alive
  reload                  Graceful reload (SIGUSR2) — fallback to restart
  restart                 Restart a process by name (or 'all')
  resurrect               Start all processes from the saved dump
  save                    Persist the current process list to ~/.pm2-go/dump.json
  show (describe, info)   Show detailed info for a process
  start                   Start a script, executable, or ecosystem file
  startup                 Generate a systemd unit to start pm2-go on boot
  stop                    Stop a process by name (or 'all')
  version (-v, --version) Print pm2-go version
  web                     Configure the public web API
```

## Build

Needs Go 1.23+:

```bash
make build      # → bin/pm2-go
make install    # → ~/.local/bin/pm2-go (override with PREFIX=)
make test
make lint       # go vet
```

## Layout

```
daemon/
├── cmd/pm2-go/main.go        CLI entry; also runs as the daemon under __daemon
└── internal/
    ├── cli/                  Command surface (start, stop, …)
    ├── daemon/               Supervisor + lifecycle + spawn/detach + watch
    ├── ipc/                  HTTP-over-Unix-socket protocol (CLI ⇄ daemon)
    ├── api/                  TCP HTTP server with bearer auth (UI ⇄ daemon)
    ├── process/              Spec + Runtime data model
    ├── logs/                 Per-process capture + tail + rotation
    ├── metrics/              /proc-based CPU/mem sampling
    ├── config/               ecosystem.config.json parser
    ├── store/                dump.json persistence
    └── paths/                Single source of truth for filesystem paths
```

The CLI auto-spawns the daemon by re-execing itself with `__daemon`, then
detaching via `Setsid`. The daemon writes its pid to `~/.pm2-go/daemon.pid`
and serves the CLI's HTTP-over-Unix-socket on `~/.pm2-go/rpc.sock`.

The web API (used by the Next.js UI in the parent directory) is **not** bound
by default. Run `pm2-go web --port 9615` once to generate
`~/.pm2-go/api-token` and `~/.pm2-go/api-port`, then `pm2-go kill && pm2-go
ping` to restart the daemon so it binds the port.

## State

```
~/.pm2-go/
├── dump.json        canonical app list (specs)
├── daemon.log       daemon stdout/stderr
├── daemon.pid       running daemon pid
├── rpc.sock         CLI ↔ daemon Unix socket
├── api-port         "<host>:<port>" for the web API
├── api-token        bearer token for the web API
├── pids/            (reserved)
└── logs/
    └── <runtime>-<out|err>.log
```

Override the base directory with `PM2_GO_HOME`.

## Ecosystem files

```json
{
  "apps": [
    {
      "name": "api",
      "script": "/srv/api/index.js",
      "interpreter": "node",
      "instances": 4,
      "namespace": "prod",
      "env": { "NODE_ENV": "production" },
      "watch": ["/srv/api/src"],
      "ignore_watch": ["*.log"],
      "max_memory_restart": "512M",
      "min_uptime": "5s",
      "exp_backoff_restart": true,
      "kill_timeout": "3s",
      "log_max_size_mb": 50,
      "log_max_backups": 5,
      "log_max_age_days": 14,
      "log_compress": true
    }
  ]
}
```

Start with `pm2-go start ecosystem.config.json`.
