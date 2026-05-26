/**
 * Typed client for the pm2-go daemon's web API.
 *
 * The daemon exposes:
 *   GET  /v1/procs                        — list runtimes (read-only snapshot)
 *   GET  /v1/specs                        — list saved app specs
 *   GET  /v1/describe?name=...            — spec + runtimes for one app
 *   POST /v1/start  (JSON body Spec)      — create or replace an app
 *   POST /v1/start?name=...               — start an existing app
 *   POST /v1/stop?name=...
 *   POST /v1/restart?name=...
 *   POST /v1/reload?name=...
 *   POST /v1/delete?name=...
 *   GET  /v1/logs?name=...&n=...&stream=  — tail
 *   GET  /v1/logs/stream?name=...         — ndjson live tail (returns ReadableStream)
 *
 * Configuration via env (server-side only):
 *   PM2_GO_DAEMON_URL    e.g. http://127.0.0.1:9615 (default)
 *   PM2_GO_DAEMON_TOKEN  bearer token (required for non-healthz endpoints)
 *
 * As a dev fallback, if PM2_GO_DAEMON_TOKEN is unset and the file
 * `~/.pm2-go/api-token` is readable, we use that. This means a single-user
 * dev box can run the daemon and the Next.js app side-by-side without
 * additional config.
 */

import { readFileSync } from "node:fs";
import { homedir } from "node:os";
import { join } from "node:path";

export type ProcessState =
  | "launching"
  | "online"
  | "stopping"
  | "stopped"
  | "errored"
  | "waiting_restart"
  | "online_restarting";

export type ProcessView = {
  id: string;
  app_id: string;
  name: string;
  instance_id: number;
  namespace: string;
  state: ProcessState;
  pid: number;
  started_at?: string;
  uptime_seconds: number;
  restarts: number;
  unstable_restarts: number;
  cpu: number;
  mem: number;
  exit_code: number;
  last_error?: string;
};

export type ProcessSpec = {
  id: string;
  name: string;
  script: string;
  args?: string[];
  interpreter?: string;
  interpreter_args?: string[];
  cwd?: string;
  env?: Record<string, string>;
  env_files?: string[];
  instances: number;
  exec_mode?: "fork" | "cluster";
  namespace: string;
  autorestart_disabled?: boolean;
  max_restarts: number;
  min_uptime?: number;
  restart_delay?: number;
  exp_backoff_restart?: boolean;
  kill_timeout: number;
  max_memory_restart?: number;
  watch?: string[];
  ignore_watch?: string[];
  log_max_size_mb?: number;
  log_max_backups?: number;
  log_max_age_days?: number;
  log_compress?: boolean;
};

let cachedToken: string | null = null;

function getToken(): string {
  if (process.env.PM2_GO_DAEMON_TOKEN) return process.env.PM2_GO_DAEMON_TOKEN;
  if (cachedToken) return cachedToken;
  try {
    const p = process.env.PM2_GO_HOME
      ? join(process.env.PM2_GO_HOME, "api-token")
      : join(homedir(), ".pm2-go", "api-token");
    cachedToken = readFileSync(p, "utf8").trim();
    return cachedToken;
  } catch {
    throw new Error(
      "pm2-go daemon token not configured. Set PM2_GO_DAEMON_TOKEN or run `pm2-go web` on the daemon host.",
    );
  }
}

function getBaseUrl(): string {
  return process.env.PM2_GO_DAEMON_URL ?? "http://127.0.0.1:9615";
}

type FetchOpts = { method?: string; body?: unknown; signal?: AbortSignal };

async function daemonFetch(path: string, opts: FetchOpts = {}): Promise<Response> {
  const url = getBaseUrl() + path;
  const init: RequestInit = {
    method: opts.method ?? "GET",
    headers: {
      Authorization: `Bearer ${getToken()}`,
      ...(opts.body ? { "Content-Type": "application/json" } : {}),
    },
    body: opts.body ? JSON.stringify(opts.body) : undefined,
    cache: "no-store",
    signal: opts.signal,
  };
  const res = await fetch(url, init);
  if (!res.ok) {
    const text = await res.text().catch(() => "");
    throw new Error(`pm2-go daemon ${res.status} ${path}: ${text || res.statusText}`);
  }
  return res;
}

export async function listProcs(): Promise<ProcessView[]> {
  const r = await daemonFetch("/v1/procs");
  const data = (await r.json()) as { procs: ProcessView[] };
  return data.procs ?? [];
}

export async function listSpecs(): Promise<ProcessSpec[]> {
  const r = await daemonFetch("/v1/specs");
  const data = (await r.json()) as { specs: ProcessSpec[] };
  return data.specs ?? [];
}

export async function describeApp(
  name: string,
): Promise<{ spec: ProcessSpec; procs: ProcessView[] }> {
  const r = await daemonFetch(`/v1/describe?name=${encodeURIComponent(name)}`);
  return (await r.json()) as { spec: ProcessSpec; procs: ProcessView[] };
}

export async function startApp(spec: Partial<ProcessSpec>): Promise<void> {
  await daemonFetch("/v1/start", { method: "POST", body: spec });
}

export async function startByName(name: string): Promise<void> {
  await daemonFetch(`/v1/start?name=${encodeURIComponent(name)}`, { method: "POST" });
}

export async function stopApp(name: string): Promise<void> {
  await daemonFetch(`/v1/stop?name=${encodeURIComponent(name)}`, { method: "POST" });
}

export async function restartApp(name: string): Promise<void> {
  await daemonFetch(`/v1/restart?name=${encodeURIComponent(name)}`, { method: "POST" });
}

export async function reloadApp(name: string): Promise<void> {
  await daemonFetch(`/v1/reload?name=${encodeURIComponent(name)}`, { method: "POST" });
}

export async function deleteApp(name: string): Promise<void> {
  await daemonFetch(`/v1/delete?name=${encodeURIComponent(name)}`, { method: "POST" });
}

export type MetricSample = { time: number; cpu: number; mem: number };

export async function metricsHistory(
  name: string,
): Promise<Record<string, MetricSample[]>> {
  const r = await daemonFetch(`/v1/metrics?name=${encodeURIComponent(name)}`);
  const data = (await r.json()) as { series: Record<string, MetricSample[]> };
  return data.series ?? {};
}

export async function saveDump(): Promise<void> {
  await daemonFetch("/v1/save", { method: "POST" });
}

export async function stopAll(): Promise<void> {
  await daemonFetch("/v1/stop-all", { method: "POST" });
}

export async function startAll(): Promise<void> {
  await daemonFetch("/v1/start-all", { method: "POST" });
}

export async function deleteAll(): Promise<void> {
  await daemonFetch("/v1/delete-all", { method: "POST" });
}

export async function tailLogs(
  name: string,
  n = 100,
  stream?: "out" | "err",
): Promise<Record<string, string[]>> {
  const params = new URLSearchParams({ name, n: String(n) });
  if (stream) params.set("stream", stream);
  const r = await daemonFetch(`/v1/logs?${params.toString()}`);
  const data = (await r.json()) as { streams: Record<string, string[]> };
  return data.streams ?? {};
}

/**
 * streamLogs returns a Response whose body is ndjson — one `{runtime, stream, time, text}` per line.
 * The route handler can pipe this body straight back to the browser.
 */
export async function streamLogs(name: string, signal?: AbortSignal): Promise<Response> {
  return daemonFetch(`/v1/logs/stream?name=${encodeURIComponent(name)}`, { signal });
}
