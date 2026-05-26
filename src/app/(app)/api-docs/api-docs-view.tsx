"use client";

import { useEffect, useState } from "react";
import { toast } from "sonner";
import { Check, Copy, Eye, EyeOff } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";

type Props = {
  baseURL: string;
  /** null = the viewer can't see the token (lacks processes:write). */
  token: string | null;
};

/**
 * Single source of truth for the TOC + section ids. Order here = order in the
 * sidebar. `methods` drives the coloured pill in front of each entry.
 */
const TOC: { id: string; title: string; methods: string[] }[] = [
  { id: "endpoint", title: "Daemon endpoint", methods: [] },
  { id: "healthz", title: "/healthz", methods: ["GET"] },
  { id: "procs", title: "/v1/procs", methods: ["GET"] },
  { id: "specs", title: "/v1/specs", methods: ["GET"] },
  { id: "describe", title: "/v1/describe", methods: ["GET"] },
  { id: "start", title: "/v1/start", methods: ["POST"] },
  { id: "stop", title: "/v1/stop", methods: ["POST"] },
  { id: "restart", title: "/v1/restart", methods: ["POST"] },
  { id: "reload", title: "/v1/reload", methods: ["POST"] },
  { id: "delete", title: "/v1/delete", methods: ["POST"] },
  { id: "bulk", title: "Bulk ops", methods: ["POST"] },
  { id: "logs", title: "/v1/logs", methods: ["GET"] },
  { id: "logs-stream", title: "/v1/logs/stream", methods: ["GET"] },
  { id: "metrics", title: "/v1/metrics", methods: ["GET"] },
  { id: "errors", title: "Errors", methods: [] },
  { id: "spec-fields", title: "Spec fields", methods: [] },
];

export function ApiDocsView({ baseURL, token }: Props) {
  const [showToken, setShowToken] = useState(false);
  const [activeId, setActiveId] = useState<string>("endpoint");
  // The example shows `$TOKEN` if the user can't see the real value.
  const tokenDisplay = token
    ? showToken
      ? token
      : token.slice(0, 6) + "…" + token.slice(-4)
    : "$TOKEN";
  const liveToken = token ?? "$TOKEN";

  // Highlight the TOC entry whose section the user last scrolled past — i.e.
  // the section currently sitting at the top of the viewport.
  //
  // Two gotchas this implementation handles:
  //
  //   1. The actual scrollable element is `<main>` (overflow-auto) from
  //      (app)/layout.tsx — NOT `window`. Listening on window does nothing
  //      because window doesn't scroll here. We walk up to find the real
  //      scroll container and bind to it.
  //   2. Hash navigation (clicking an <a href="#reload">) may not fire a
  //      scroll event if the target is already in the visible area — bind
  //      `hashchange` too so the active id updates immediately.
  useEffect(() => {
    // Sections have `scroll-mt-20` (80px) and the topbar is `h-14` (56px), so
    // after a hash click the target section's getBoundingClientRect().top
    // lands around 136. The trigger must be at least that high or the *next*
    // section never registers as active. Add a small headroom.
    const TRIGGER_OFFSET = 150; // px from viewport top
    let raf = 0;

    const update = () => {
      raf = 0;
      let current = TOC[0]?.id ?? "";
      for (const t of TOC) {
        const el = document.getElementById(t.id);
        if (!el) continue;
        const top = el.getBoundingClientRect().top;
        if (top - TRIGGER_OFFSET <= 0) {
          current = t.id;
        } else {
          break;
        }
      }
      setActiveId(current);
    };
    const schedule = () => {
      if (raf === 0) raf = requestAnimationFrame(update);
    };

    // Find every scrollable ancestor of the first section. There may be more
    // than one if the layout nests scrollable boxes.
    const scrollers: (Window | HTMLElement)[] = [window];
    const first = document.getElementById(TOC[0]?.id ?? "");
    if (first) {
      let p: HTMLElement | null = first.parentElement;
      while (p) {
        const overflowY = getComputedStyle(p).overflowY;
        if (overflowY === "auto" || overflowY === "scroll") scrollers.push(p);
        p = p.parentElement;
      }
    }

    for (const s of scrollers) s.addEventListener("scroll", schedule, { passive: true } as AddEventListenerOptions);
    window.addEventListener("resize", schedule);
    window.addEventListener("hashchange", schedule);
    update();
    return () => {
      if (raf !== 0) cancelAnimationFrame(raf);
      for (const s of scrollers) s.removeEventListener("scroll", schedule);
      window.removeEventListener("resize", schedule);
      window.removeEventListener("hashchange", schedule);
    };
  }, []);

  return (
    <div className="grid gap-6 lg:grid-cols-[240px_1fr]">
      <Toc activeId={activeId} onPick={setActiveId} />
      <div className="space-y-6">
      <Card id="endpoint" className="scroll-mt-20">
        <CardHeader>
          <CardTitle className="text-base">Daemon endpoint</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          <Field label="Base URL" value={baseURL} />
          <Field
            label="Auth token"
            value={tokenDisplay}
            actions={
              token ? (
                <>
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon"
                    title={showToken ? "Hide" : "Reveal"}
                    onClick={() => setShowToken((v) => !v)}
                  >
                    {showToken ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                  </Button>
                  <CopyButton value={token} />
                </>
              ) : (
                <span className="text-xs text-muted-foreground">
                  requires <code className="font-mono">processes:write</code>
                </span>
              )
            }
          />
          <p className="text-xs text-muted-foreground">
            Every <code className="font-mono">/v1/*</code> request must include{" "}
            <code className="font-mono">Authorization: Bearer &lt;token&gt;</code>. Use{" "}
            <code className="font-mono">pm2-go web --regenerate-token</code> on the daemon host to
            rotate.
          </p>
        </CardContent>
      </Card>

      <Section id="healthz" title="GET /healthz" methods={["GET"]} sub="liveness probe, no auth">
        <Curl baseURL={baseURL} token={liveToken} noAuth>
          {`curl -s ${baseURL}/healthz`}
        </Curl>
        <Response>{`{"ok":true}`}</Response>
      </Section>

      <Section
        id="procs"
        title="GET /v1/procs"
        methods={["GET"]}
        sub="every runtime instance — one row per cluster instance"
      >
        <Curl baseURL={baseURL} token={liveToken}>
          {`curl -sH "Authorization: Bearer ${liveToken}" ${baseURL}/v1/procs | jq`}
        </Curl>
        <Response>{`{
  "procs": [{
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
  }]
}`}</Response>
        <p className="text-xs text-muted-foreground">
          <code className="font-mono">state</code> ∈ {"{"}launching, online, stopping, stopped,
          errored, waiting_restart, online_restarting{"}"}
        </p>
      </Section>

      <Section
        id="specs"
        title="GET /v1/specs"
        methods={["GET"]}
        sub="persisted spec list (one row per app, not per runtime)"
      >
        <Curl baseURL={baseURL} token={liveToken}>
          {`curl -sH "Authorization: Bearer ${liveToken}" ${baseURL}/v1/specs | jq`}
        </Curl>
      </Section>

      <Section
        id="describe"
        title="GET /v1/describe?name=…"
        methods={["GET"]}
        sub="spec + runtimes for a single app"
      >
        <Curl baseURL={baseURL} token={liveToken}>
          {`curl -sH "Authorization: Bearer ${liveToken}" "${baseURL}/v1/describe?name=api"`}
        </Curl>
        <Response>{`{ "spec": { … }, "procs": [ … ] }`}</Response>
        <p className="text-xs text-muted-foreground">404 if name doesn't exist.</p>
      </Section>

      <Section
        id="start"
        title="POST /v1/start"
        methods={["POST"]}
        sub="add or replace an app from a JSON spec, or start an existing one"
      >
        <h4 className="text-xs font-semibold uppercase text-muted-foreground">From a spec body</h4>
        <Curl baseURL={baseURL} token={liveToken}>{`curl -sH "Authorization: Bearer ${liveToken}" \\
  -H "Content-Type: application/json" \\
  -X POST ${baseURL}/v1/start \\
  -d '{
    "name": "api",
    "script": "/srv/api/index.js",
    "interpreter": "node",
    "instances": 4,
    "namespace": "prod",
    "env": { "NODE_ENV": "production" },
    "watch": ["/srv/api/src"],
    "max_memory_restart": 314572800
  }'`}</Curl>

        <h4 className="text-xs font-semibold uppercase text-muted-foreground">Existing app, by name</h4>
        <Curl baseURL={baseURL} token={liveToken}>
          {`curl -sH "Authorization: Bearer ${liveToken}" -X POST "${baseURL}/v1/start?name=api"`}
        </Curl>
        <p className="text-xs text-muted-foreground">
          If a spec with the given <code className="font-mono">name</code> already exists, it's
          stopped and re-spawned with the new config. Logs on disk are preserved.
        </p>
      </Section>

      <Section
        id="stop"
        title="POST /v1/stop?name=…"
        methods={["POST"]}
        sub="SIGTERM → SIGKILL after kill_timeout"
      >
        <Curl baseURL={baseURL} token={liveToken}>
          {`curl -sH "Authorization: Bearer ${liveToken}" -X POST "${baseURL}/v1/stop?name=api"`}
        </Curl>
      </Section>

      <Section
        id="restart"
        title="POST /v1/restart?name=…"
        methods={["POST"]}
        sub="stop then start; restart counter bumps by one"
      >
        <Curl baseURL={baseURL} token={liveToken}>
          {`curl -sH "Authorization: Bearer ${liveToken}" -X POST "${baseURL}/v1/restart?name=api"`}
        </Curl>
      </Section>

      <Section
        id="reload"
        title="POST /v1/reload?name=…"
        methods={["POST"]}
        sub="SIGUSR2 to every instance — falls back to a hard restart for apps that don't handle it"
      >
        <Curl baseURL={baseURL} token={liveToken}>
          {`curl -sH "Authorization: Bearer ${liveToken}" -X POST "${baseURL}/v1/reload?name=api"`}
        </Curl>
      </Section>

      <Section
        id="delete"
        title="POST /v1/delete?name=…"
        methods={["POST"]}
        sub="stop + remove from dump (log files on disk are kept)"
      >
        <Curl baseURL={baseURL} token={liveToken}>
          {`curl -sH "Authorization: Bearer ${liveToken}" -X POST "${baseURL}/v1/delete?name=api"`}
        </Curl>
      </Section>

      <Section
        id="bulk"
        title="Bulk operations"
        methods={["POST"]}
        sub="POST /v1/save | start-all | stop-all | delete-all — no body"
      >
        <Curl baseURL={baseURL} token={liveToken}>{`curl -sH "Authorization: Bearer ${liveToken}" -X POST ${baseURL}/v1/save
curl -sH "Authorization: Bearer ${liveToken}" -X POST ${baseURL}/v1/start-all
curl -sH "Authorization: Bearer ${liveToken}" -X POST ${baseURL}/v1/stop-all
curl -sH "Authorization: Bearer ${liveToken}" -X POST ${baseURL}/v1/delete-all`}</Curl>
      </Section>

      <Section
        id="logs"
        title="GET /v1/logs?name=…&n=…&stream=out|err"
        methods={["GET"]}
        sub="recent N lines from stdout/stderr per runtime"
      >
        <Curl baseURL={baseURL} token={liveToken}>
          {`curl -sH "Authorization: Bearer ${liveToken}" "${baseURL}/v1/logs?name=api&n=20" | jq`}
        </Curl>
        <Response>{`{
  "streams": {
    "api-0:out": ["listening on :3000", "got request /healthz"],
    "api-0:err": []
  }
}`}</Response>
        <p className="text-xs text-muted-foreground">
          Empty streams are always <code className="font-mono">[]</code>, never null.
        </p>
      </Section>

      <Section
        id="logs-stream"
        title="GET /v1/logs/stream?name=…"
        methods={["GET"]}
        sub="ndjson live tail — one JSON object per log line, server flushes on every write"
      >
        <Curl baseURL={baseURL} token={liveToken}>
          {`curl -NsH "Authorization: Bearer ${liveToken}" "${baseURL}/v1/logs/stream?name=api"`}
        </Curl>
        <Response>{`{"runtime":"api-0","stream":"out","time":1779781146174,"text":"listening on :3000\\n"}
{"runtime":"api-0","stream":"err","time":1779781146210,"text":"warning: fd leak\\n"}`}</Response>
        <p className="text-xs text-muted-foreground">
          <code className="font-mono">time</code> is unix milliseconds. Slow clients drop messages
          (per-subscriber buffer of 64).
        </p>
      </Section>

      <Section
        id="metrics"
        title="GET /v1/metrics?name=…"
        methods={["GET"]}
        sub="rolling ~1Hz CPU/mem window per runtime, last 5 min"
      >
        <Curl baseURL={baseURL} token={liveToken}>
          {`curl -sH "Authorization: Bearer ${liveToken}" "${baseURL}/v1/metrics?name=api" | jq`}
        </Curl>
        <Response>{`{
  "series": {
    "api-0": [
      { "time": 1779781140000, "cpu": 4.2, "mem": 39845888 },
      { "time": 1779781141000, "cpu": 5.1, "mem": 39845888 }
    ]
  }
}`}</Response>
        <p className="text-xs text-muted-foreground">
          CPU is percent of one core. Mem is bytes (RSS).
        </p>
      </Section>

      <Section id="errors" title="Errors" methods={[]} sub="JSON error envelope, status code carries the meaning">
        <Response>{`{ "error": "<message>" }`}</Response>
        <table className="w-full text-sm">
          <thead className="border-b border-border text-left text-xs uppercase text-muted-foreground">
            <tr>
              <th className="py-2 pr-3">Status</th>
              <th className="py-2 pr-3">Meaning</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-border">
            {[
              ["200", "OK"],
              ["400", "Malformed request (bad JSON, missing query param)"],
              ["401", "Missing / wrong bearer token"],
              ["404", "Named app does not exist on the daemon"],
              ["502", "Internal error (propagated as { error: ... })"],
            ].map(([code, desc]) => (
              <tr key={code}>
                <td className="py-2 pr-3 font-mono">{code}</td>
                <td className="py-2 pr-3 text-muted-foreground">{desc}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </Section>

      <Section id="spec-fields" title="Spec fields" methods={[]} sub="every field accepted by POST /v1/start">
        <table className="w-full text-sm">
          <thead className="border-b border-border text-left text-xs uppercase text-muted-foreground">
            <tr>
              <th className="py-2 pr-3">Field</th>
              <th className="py-2 pr-3">Type</th>
              <th className="py-2 pr-3">Notes</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-border">
            {SPEC_FIELDS.map((f) => (
              <tr key={f.name}>
                <td className="py-2 pr-3 font-mono">{f.name}</td>
                <td className="py-2 pr-3 text-muted-foreground">{f.type}</td>
                <td className="py-2 pr-3 text-muted-foreground">{f.notes}</td>
              </tr>
            ))}
          </tbody>
        </table>
        <p className="text-xs text-muted-foreground">
          Durations are nanoseconds (Go <code className="font-mono">time.Duration</code> JSON
          encoding). <code className="font-mono">max_memory_restart</code> is bytes.
        </p>
      </Section>
      </div>
    </div>
  );
}

function Toc({ activeId, onPick }: { activeId: string; onPick: (id: string) => void }) {
  return (
    <aside className="order-first lg:sticky lg:top-20 lg:self-start">
      <Card>
        <CardHeader>
          <CardTitle className="text-xs uppercase tracking-wide text-muted-foreground">
            On this page
          </CardTitle>
        </CardHeader>
        <CardContent>
          <nav>
            <ul className="space-y-px">
              {TOC.map((t) => (
                <li key={t.id}>
                  <a
                    href={`#${t.id}`}
                    onClick={() => onPick(t.id)}
                    className={`flex items-center gap-2 rounded-md px-2 py-1 text-xs transition-colors ${
                      activeId === t.id
                        ? "bg-accent font-medium text-accent-foreground"
                        : "text-muted-foreground hover:bg-accent/60 hover:text-foreground"
                    }`}
                  >
                    {t.methods.length > 0 ? (
                      <span
                        className={`inline-block w-10 shrink-0 rounded text-[10px] font-medium uppercase ${
                          t.methods[0] === "GET"
                            ? "text-emerald-600 dark:text-emerald-400"
                            : "text-amber-600 dark:text-amber-400"
                        }`}
                      >
                        {t.methods[0]}
                      </span>
                    ) : (
                      <span className="inline-block w-10 shrink-0" aria-hidden />
                    )}
                    <span className="truncate font-mono">{t.title}</span>
                  </a>
                </li>
              ))}
            </ul>
          </nav>
        </CardContent>
      </Card>
    </aside>
  );
}

function Section({
  id,
  title,
  methods,
  sub,
  children,
}: {
  id: string;
  title: string;
  methods: string[];
  sub: string;
  children: React.ReactNode;
}) {
  return (
    <Card id={id} className="scroll-mt-20">
      <CardHeader>
        <CardTitle className="flex flex-wrap items-center gap-2 text-base">
          {methods.map((m) => (
            <Badge key={m} variant="outline" className="font-mono">
              {m}
            </Badge>
          ))}
          <code className="font-mono text-sm">{title}</code>
        </CardTitle>
        <p className="text-xs text-muted-foreground">{sub}</p>
      </CardHeader>
      <CardContent className="space-y-3">{children}</CardContent>
    </Card>
  );
}

function Field({
  label,
  value,
  actions,
}: {
  label: string;
  value: string;
  actions?: React.ReactNode;
}) {
  return (
    <div className="flex items-center justify-between gap-3 rounded-md border border-border bg-muted/30 p-2">
      <div className="min-w-0 flex-1">
        <p className="text-xs uppercase text-muted-foreground">{label}</p>
        <p className="truncate font-mono text-sm">{value}</p>
      </div>
      <div className="flex shrink-0 items-center gap-1">{actions ?? <CopyButton value={value} />}</div>
    </div>
  );
}

function CopyButton({ value }: { value: string }) {
  const [copied, setCopied] = useState(false);
  return (
    <Button
      type="button"
      variant="ghost"
      size="icon"
      title="Copy to clipboard"
      onClick={async () => {
        try {
          await navigator.clipboard.writeText(value);
          setCopied(true);
          toast.success("Copied");
          setTimeout(() => setCopied(false), 1200);
        } catch (e) {
          toast.error(`Copy failed: ${(e as Error).message}`);
        }
      }}
    >
      {copied ? <Check className="h-4 w-4" /> : <Copy className="h-4 w-4" />}
    </Button>
  );
}

function Curl({
  children,
  baseURL: _baseURL,
  token: _token,
  noAuth: _noAuth,
}: {
  children: string;
  baseURL: string;
  token: string;
  noAuth?: boolean;
}) {
  return (
    <div className="relative">
      <pre className="overflow-x-auto rounded-md bg-zinc-950 p-3 pr-12 text-xs leading-relaxed text-zinc-100">
        <code>{children}</code>
      </pre>
      <div className="absolute right-2 top-2">
        <CopyButton value={children} />
      </div>
    </div>
  );
}

function Response({ children }: { children: string }) {
  return (
    <div className="relative">
      <pre className="overflow-x-auto rounded-md bg-muted p-3 pr-12 text-xs leading-relaxed">
        <code>{children}</code>
      </pre>
      <div className="absolute right-2 top-2">
        <CopyButton value={children} />
      </div>
    </div>
  );
}

const SPEC_FIELDS: { name: string; type: string; notes: string }[] = [
  { name: "name", type: "string", notes: "Required. Primary key." },
  { name: "script", type: "string", notes: "Required. Absolute path or PATH lookup." },
  { name: "args", type: "string[]", notes: "Args passed to script." },
  { name: "interpreter", type: "string", notes: "node, python3, /bin/bash. Empty = direct exec." },
  { name: "interpreter_args", type: "string[]", notes: "Args before script." },
  { name: "cwd", type: "string", notes: "Working directory." },
  { name: "env", type: "map<string,string>", notes: "Wins over env_files." },
  { name: "env_files", type: "string[]", notes: ".env file paths." },
  { name: "instances", type: "int", notes: "Default 1. >1 = cluster." },
  { name: "exec_mode", type: '"fork"|"cluster"', notes: "Only fork is implemented." },
  { name: "namespace", type: "string", notes: "Filter label. Default 'default'." },
  { name: "autorestart_disabled", type: "bool", notes: "true = don't restart on exit." },
  { name: "max_restarts", type: "int", notes: "Cap unstable restarts. 0 = unlimited. Default 16." },
  { name: "min_uptime", type: "int (ns)", notes: "Below this = unstable. Default 1s." },
  { name: "restart_delay", type: "int (ns)", notes: "Sleep between attempts." },
  { name: "exp_backoff_restart", type: "bool", notes: "Exponential backoff (capped 64s)." },
  { name: "kill_timeout", type: "int (ns)", notes: "SIGTERM → SIGKILL grace. Default 1.6s." },
  { name: "max_memory_restart", type: "uint64", notes: "Restart if RSS exceeds (bytes)." },
  { name: "stop_exit_codes", type: "int[]", notes: "Don't restart for these exits." },
  { name: "watch", type: "string[]", notes: "fsnotify watch (debounced restart)." },
  { name: "ignore_watch", type: "string[]", notes: "Globs vs basename. dotfiles + node_modules always skipped." },
  { name: "log_max_size_mb", type: "int", notes: "Enable lumberjack rotation at this size." },
  { name: "log_max_backups", type: "int", notes: "Lumberjack." },
  { name: "log_max_age_days", type: "int", notes: "Lumberjack." },
  { name: "log_compress", type: "bool", notes: "Lumberjack gzip rotated files." },
];
