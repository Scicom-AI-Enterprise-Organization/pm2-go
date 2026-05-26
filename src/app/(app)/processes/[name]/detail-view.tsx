"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import type { ProcessSpec, ProcessView } from "@/lib/pm2";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { FileText, Pencil } from "lucide-react";
import { humanBytes, humanDuration, stateColor } from "../format";
import { ProcessRowActions } from "../row-actions";
import { MetricsChart } from "./metrics-chart";

const POLL_MS = 2000;

type Props = {
  name: string;
  initial: { spec: ProcessSpec; procs: ProcessView[] };
  canWrite: boolean;
  canDelete: boolean;
  canLogs: boolean;
};

/**
 * Client-side detail panel that re-polls /api/pm2/describe every 2s so PID,
 * state, uptime, and metrics stay live without a page reload.
 */
export function DetailView({ name, initial, canWrite, canDelete, canLogs }: Props) {
  const [data, setData] = useState(initial);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    let timer: ReturnType<typeof setTimeout> | null = null;
    async function tick() {
      try {
        const r = await fetch(`/api/pm2/describe?name=${encodeURIComponent(name)}`, {
          cache: "no-store",
        });
        if (!r.ok) {
          const body = (await r.json().catch(() => ({}))) as { error?: string };
          if (!cancelled) setError(body.error ?? r.statusText);
          return;
        }
        const body = (await r.json()) as { spec: ProcessSpec; procs: ProcessView[] };
        if (!cancelled) {
          setData(body);
          setError(null);
        }
      } catch (e) {
        if (!cancelled) setError((e as Error).message);
      } finally {
        if (!cancelled) timer = setTimeout(tick, POLL_MS);
      }
    }
    timer = setTimeout(tick, POLL_MS);
    return () => {
      cancelled = true;
      if (timer) clearTimeout(timer);
    };
  }, [name]);

  const { spec, procs } = data;

  return (
    <div className="space-y-6">
      <div className="flex items-start justify-between gap-4">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">{spec.name}</h1>
          <p className="mt-1 font-mono text-xs text-muted-foreground">
            {spec.interpreter ? `${spec.interpreter} ` : ""}
            {spec.script}
            {spec.args && spec.args.length > 0 ? ` ${spec.args.join(" ")}` : ""}
          </p>
        </div>
        <div className="flex gap-2">
          {canLogs ? (
            <Link href={`/processes/${encodeURIComponent(spec.name)}/logs`}>
              <Button variant="outline">
                <FileText className="h-4 w-4" /> Live logs
              </Button>
            </Link>
          ) : null}
          {canWrite ? (
            <Link href={`/processes/${encodeURIComponent(spec.name)}/edit`}>
              <Button>
                <Pencil className="h-4 w-4" /> Edit
              </Button>
            </Link>
          ) : null}
        </div>
      </div>

      {error ? (
        <Card className="border-red-200 dark:border-red-900">
          <CardHeader>
            <CardTitle className="text-red-700 dark:text-red-300">Daemon unreachable</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-sm text-muted-foreground">{error}</p>
          </CardContent>
        </Card>
      ) : null}

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard label="Namespace" value={spec.namespace} />
        <StatCard label="Instances" value={String(spec.instances)} />
        <StatCard label="Autorestart" value={spec.autorestart_disabled ? "off" : "on"} />
        <StatCard
          label="Max memory restart"
          value={spec.max_memory_restart ? humanBytes(spec.max_memory_restart) : "—"}
        />
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            Instances <Badge variant="secondary">{procs.length}</Badge>
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead className="border-b border-border text-left text-xs uppercase text-muted-foreground">
                <tr>
                  <th className="py-2 pr-3">Runtime</th>
                  <th className="py-2 pr-3">PID</th>
                  <th className="py-2 pr-3">State</th>
                  <th className="py-2 pr-3">Uptime</th>
                  <th className="py-2 pr-3">↻</th>
                  <th className="py-2 pr-3">CPU</th>
                  <th className="py-2 pr-3">Mem</th>
                  <th className="py-2" />
                </tr>
              </thead>
              <tbody className="divide-y divide-border">
                {procs.map((p) => (
                  <tr key={p.id}>
                    <td className="py-2 pr-3 font-mono">{p.id}</td>
                    <td className="py-2 pr-3 font-mono text-muted-foreground">
                      {p.pid || "—"}
                    </td>
                    <td className="py-2 pr-3">
                      <span className={`rounded px-2 py-0.5 text-xs font-medium ${stateColor(p.state)}`}>
                        {p.state}
                      </span>
                    </td>
                    <td className="py-2 pr-3 text-muted-foreground">
                      {humanDuration(p.uptime_seconds)}
                    </td>
                    <td className="py-2 pr-3 text-muted-foreground">{p.restarts}</td>
                    <td className="py-2 pr-3 text-muted-foreground">{p.cpu.toFixed(1)}%</td>
                    <td className="py-2 pr-3 text-muted-foreground">{humanBytes(p.mem)}</td>
                    <td className="py-2">
                      <ProcessRowActions
                        name={spec.name}
                        state={p.state}
                        canWrite={canWrite}
                        canDelete={canDelete}
                      />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>CPU &amp; memory</CardTitle>
        </CardHeader>
        <CardContent>
          <MetricsChart name={spec.name} />
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Configuration</CardTitle>
        </CardHeader>
        <CardContent>
          <dl className="grid grid-cols-1 gap-3 text-sm sm:grid-cols-2">
            <Detail label="Script" value={spec.script} mono />
            {spec.interpreter ? (
              <Detail label="Interpreter" value={spec.interpreter} mono />
            ) : null}
            {spec.cwd ? <Detail label="CWD" value={spec.cwd} mono /> : null}
            {spec.args && spec.args.length > 0 ? (
              <Detail label="Args" value={spec.args.join(" ")} mono />
            ) : null}
            {spec.watch && spec.watch.length > 0 ? (
              <Detail label="Watch" value={spec.watch.join(", ")} mono />
            ) : null}
            <Detail label="Kill timeout" value={`${(spec.kill_timeout / 1e9).toFixed(2)}s`} />
            <Detail label="Max restarts" value={String(spec.max_restarts)} />
            {spec.env && Object.keys(spec.env).length > 0 ? (
              <Detail
                label="Env"
                mono
                value={Object.entries(spec.env)
                  .map(([k, v]) => `${k}=${v}`)
                  .join("\n")}
              />
            ) : null}
          </dl>
        </CardContent>
      </Card>
    </div>
  );
}

function StatCard({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-lg border border-border bg-card p-4">
      <p className="text-xs uppercase text-muted-foreground">{label}</p>
      <p className="mt-1 text-xl font-semibold">{value}</p>
    </div>
  );
}

function Detail({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
  return (
    <div>
      <dt className="text-xs uppercase text-muted-foreground">{label}</dt>
      <dd className={`mt-0.5 whitespace-pre-wrap ${mono ? "font-mono text-xs" : ""}`}>
        {value}
      </dd>
    </div>
  );
}
