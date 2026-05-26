"use client";

import Link from "next/link";
import { useEffect, useMemo, useState } from "react";
import {
  Activity,
  AlertTriangle,
  Cpu,
  HardDrive,
  Plus,
  Upload,
  Zap,
} from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import type { ProcessView } from "@/lib/pm2";

const POLL_MS = 5000;

type Props = {
  initial: ProcessView[];
  canWrite: boolean;
  daemonReachable: boolean;
  daemonError?: string;
};

export function DashboardView({ initial, canWrite, daemonReachable, daemonError }: Props) {
  const [procs, setProcs] = useState<ProcessView[]>(initial);
  const [reachable, setReachable] = useState(daemonReachable);
  const [errorMsg, setErrorMsg] = useState<string | null>(daemonError ?? null);

  useEffect(() => {
    let cancelled = false;
    let timer: ReturnType<typeof setTimeout> | null = null;
    async function tick() {
      try {
        const r = await fetch("/api/pm2/procs", { cache: "no-store" });
        const data = (await r.json()) as { procs: ProcessView[]; error?: string };
        if (cancelled) return;
        if (data.error) {
          setReachable(false);
          setErrorMsg(data.error);
        } else {
          setReachable(true);
          setErrorMsg(null);
          setProcs(data.procs ?? []);
        }
      } catch (e) {
        if (!cancelled) {
          setReachable(false);
          setErrorMsg((e as Error).message);
        }
      } finally {
        if (!cancelled) timer = setTimeout(tick, POLL_MS);
      }
    }
    timer = setTimeout(tick, POLL_MS);
    return () => {
      cancelled = true;
      if (timer) clearTimeout(timer);
    };
  }, []);

  const stats = useMemo(() => deriveStats(procs), [procs]);

  return (
    <div className="space-y-6">
      {!reachable ? (
        <Card className="border-red-200 dark:border-red-900">
          <CardContent className="flex items-start gap-3 pt-6">
            <AlertTriangle className="mt-0.5 h-5 w-5 shrink-0 text-red-600 dark:text-red-400" />
            <div>
              <p className="text-sm font-medium text-red-700 dark:text-red-300">
                Daemon unreachable
              </p>
              <p className="mt-1 text-xs text-muted-foreground">
                {errorMsg ??
                  "The Next.js server can't reach the pm2-go daemon. Check PM2_GO_DAEMON_URL / PM2_GO_DAEMON_TOKEN, or `docker compose logs daemon`."}
              </p>
            </div>
          </CardContent>
        </Card>
      ) : null}

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard label="Total instances" value={String(stats.total)} icon={<Activity className="h-5 w-5" />} />
        <StatCard
          label="Online"
          value={`${stats.byState.online ?? 0}`}
          sub={`${stats.total > 0 ? Math.round(((stats.byState.online ?? 0) / stats.total) * 100) : 0}%`}
          tone={(stats.byState.online ?? 0) === stats.total ? "ok" : "warn"}
          icon={<Zap className="h-5 w-5" />}
        />
        <StatCard label="CPU (sum)" value={`${stats.cpuSum.toFixed(1)}%`} icon={<Cpu className="h-5 w-5" />} />
        <StatCard label="Memory (sum)" value={humanBytes(stats.memSum)} icon={<HardDrive className="h-5 w-5" />} />
      </div>

      <div className="grid gap-4 lg:grid-cols-3">
        <Card className="lg:col-span-1">
          <CardHeader>
            <CardTitle className="text-sm font-medium">State breakdown</CardTitle>
          </CardHeader>
          <CardContent>
            {Object.keys(stats.byState).length === 0 ? (
              <p className="text-sm text-muted-foreground">No processes yet.</p>
            ) : (
              <ul className="space-y-2 text-sm">
                {Object.entries(stats.byState)
                  .sort(([, a], [, b]) => b - a)
                  .map(([state, count]) => (
                    <li key={state} className="flex items-center justify-between">
                      <span className={`rounded px-2 py-0.5 text-xs font-medium ${stateColor(state)}`}>
                        {state}
                      </span>
                      <span className="font-mono text-muted-foreground">{count}</span>
                    </li>
                  ))}
              </ul>
            )}
          </CardContent>
        </Card>

        <Card className="lg:col-span-1">
          <CardHeader>
            <CardTitle className="text-sm font-medium">Top CPU</CardTitle>
          </CardHeader>
          <CardContent>
            <TopList rows={stats.topCPU} format={(p) => `${p.cpu.toFixed(1)}%`} />
          </CardContent>
        </Card>

        <Card className="lg:col-span-1">
          <CardHeader>
            <CardTitle className="text-sm font-medium">Top memory</CardTitle>
          </CardHeader>
          <CardContent>
            <TopList rows={stats.topMem} format={(p) => humanBytes(p.mem)} />
          </CardContent>
        </Card>
      </div>

      <div className="grid gap-4 lg:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle className="text-sm font-medium">Namespaces</CardTitle>
          </CardHeader>
          <CardContent>
            {Object.keys(stats.byNamespace).length === 0 ? (
              <p className="text-sm text-muted-foreground">No processes yet.</p>
            ) : (
              <ul className="space-y-2 text-sm">
                {Object.entries(stats.byNamespace)
                  .sort(([a], [b]) => a.localeCompare(b))
                  .map(([ns, count]) => (
                    <li key={ns} className="flex items-center justify-between">
                      <Link
                        href={`/processes?ns=${encodeURIComponent(ns)}`}
                        className="font-mono hover:underline"
                      >
                        {ns}
                      </Link>
                      <span className="font-mono text-muted-foreground">{count}</span>
                    </li>
                  ))}
              </ul>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="text-sm font-medium">Quick actions</CardTitle>
          </CardHeader>
          <CardContent className="grid grid-cols-2 gap-2 text-sm">
            <QuickLink href="/processes" label="All processes" />
            {canWrite ? (
              <>
                <QuickLink href="/processes/new" label="New process" icon={<Plus className="h-4 w-4" />} />
                <QuickLink href="/processes/import" label="Import ecosystem" icon={<Upload className="h-4 w-4" />} />
              </>
            ) : null}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}

function StatCard({
  label,
  value,
  sub,
  icon,
  tone = "default",
}: {
  label: string;
  value: string;
  sub?: string;
  icon?: React.ReactNode;
  tone?: "default" | "ok" | "warn";
}) {
  const toneClass =
    tone === "ok"
      ? "text-emerald-600 dark:text-emerald-400"
      : tone === "warn"
        ? "text-amber-600 dark:text-amber-400"
        : "text-foreground";
  return (
    <Card>
      <CardContent className="pt-6">
        <div className="flex items-start justify-between">
          <div>
            <p className="text-xs uppercase text-muted-foreground">{label}</p>
            <p className={`mt-1 text-2xl font-semibold ${toneClass}`}>{value}</p>
            {sub ? <p className="mt-0.5 text-xs text-muted-foreground">{sub}</p> : null}
          </div>
          {icon ? <span className="text-muted-foreground">{icon}</span> : null}
        </div>
      </CardContent>
    </Card>
  );
}

function TopList({ rows, format }: { rows: ProcessView[]; format: (p: ProcessView) => string }) {
  if (rows.length === 0) {
    return <p className="text-sm text-muted-foreground">No samples yet.</p>;
  }
  return (
    <ul className="space-y-1.5 text-sm">
      {rows.map((p) => (
        <li key={p.id} className="flex items-center justify-between gap-2">
          <Link
            href={`/processes/${encodeURIComponent(p.name)}`}
            className="truncate font-mono text-xs hover:underline"
          >
            {p.id}
          </Link>
          <span className="shrink-0 font-mono text-xs text-muted-foreground">{format(p)}</span>
        </li>
      ))}
    </ul>
  );
}

function QuickLink({ href, label, icon }: { href: string; label: string; icon?: React.ReactNode }) {
  return (
    <Link
      href={href}
      className="flex items-center gap-2 rounded-md border border-border px-3 py-2 transition-colors hover:bg-accent hover:text-accent-foreground"
    >
      {icon ?? <Activity className="h-4 w-4" />}
      <span>{label}</span>
    </Link>
  );
}

type Stats = {
  total: number;
  cpuSum: number;
  memSum: number;
  byState: Record<string, number>;
  byNamespace: Record<string, number>;
  topCPU: ProcessView[];
  topMem: ProcessView[];
};

function deriveStats(procs: ProcessView[]): Stats {
  const s: Stats = {
    total: procs.length,
    cpuSum: 0,
    memSum: 0,
    byState: {},
    byNamespace: {},
    topCPU: [],
    topMem: [],
  };
  for (const p of procs) {
    s.cpuSum += p.cpu ?? 0;
    s.memSum += p.mem ?? 0;
    s.byState[p.state] = (s.byState[p.state] ?? 0) + 1;
    s.byNamespace[p.namespace] = (s.byNamespace[p.namespace] ?? 0) + 1;
  }
  const onlyOnline = procs.filter((p) => p.state === "online");
  s.topCPU = [...onlyOnline].sort((a, b) => b.cpu - a.cpu).slice(0, 5);
  s.topMem = [...onlyOnline].sort((a, b) => b.mem - a.mem).slice(0, 5);
  return s;
}

function stateColor(state: string): string {
  switch (state) {
    case "online":
      return "bg-emerald-500/10 text-emerald-700 dark:text-emerald-300";
    case "launching":
    case "waiting_restart":
    case "online_restarting":
      return "bg-amber-500/10 text-amber-700 dark:text-amber-300";
    case "stopping":
      return "bg-orange-500/10 text-orange-700 dark:text-orange-300";
    case "stopped":
      return "bg-zinc-500/10 text-zinc-700 dark:text-zinc-300";
    case "errored":
      return "bg-red-500/10 text-red-700 dark:text-red-300";
    default:
      return "bg-zinc-500/10";
  }
}

function humanBytes(n: number): string {
  if (!n) return "—";
  const k = 1024;
  if (n < k) return `${n} B`;
  if (n < k * k) return `${(n / k).toFixed(1)} KB`;
  if (n < k * k * k) return `${(n / (k * k)).toFixed(1)} MB`;
  return `${(n / (k * k * k)).toFixed(2)} GB`;
}

// Account fallback used when the viewer has no processes:read.
export function AccountOnlyDashboard({
  roles,
  permissions,
  email,
  name,
}: {
  roles: string[];
  permissions: string[];
  email?: string | null;
  name?: string | null;
}) {
  return (
    <div className="space-y-6">
      <div>
        <p className="text-sm text-muted-foreground">
          You don't have <code className="font-mono">processes:read</code> yet — ask an admin to
          assign you the <Badge variant="secondary">pm2-operator</Badge> or{" "}
          <Badge variant="secondary">admin</Badge> role to see fleet stats.
        </p>
      </div>
      <div className="grid gap-4 md:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle className="text-sm font-medium">Roles</CardTitle>
          </CardHeader>
          <CardContent className="flex flex-wrap gap-2">
            {roles.length === 0 ? (
              <span className="text-sm text-muted-foreground">None assigned.</span>
            ) : (
              roles.map((r) => (
                <Badge key={r} variant="secondary">
                  {r}
                </Badge>
              ))
            )}
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle className="text-sm font-medium">Permissions</CardTitle>
          </CardHeader>
          <CardContent className="flex flex-wrap gap-2">
            {permissions.length === 0 ? (
              <span className="text-sm text-muted-foreground">None.</span>
            ) : (
              permissions.map((p) => (
                <Badge key={p} variant="outline">
                  {p}
                </Badge>
              ))
            )}
          </CardContent>
        </Card>
      </div>
      <p className="text-xs text-muted-foreground">
        Signed in as {name ?? email ?? "unknown"} {email ? `(${email})` : ""}
      </p>
    </div>
  );
}
