"use client";

import Link from "next/link";
import { useEffect, useMemo, useState, useTransition } from "react";
import { toast } from "sonner";
import type { ProcessView } from "@/lib/pm2";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Plus, RefreshCcw, Save, Upload, Pause, Play, Trash2 } from "lucide-react";
import { humanBytes, humanDuration, stateColor } from "./format";
import { ProcessRowActions } from "./row-actions";
import {
  deleteAllAction,
  saveDumpAction,
  startAllAction,
  stopAllAction,
} from "./actions";
import { ConfirmDeleteDialog } from "./confirm-delete-dialog";

type Perms = {
  canWrite: boolean;
  canDelete: boolean;
};

type Props = {
  initial: ProcessView[];
} & Perms;

const POLL_MS = 2000;

export function ProcessesView({ initial, canWrite, canDelete }: Props) {
  const [procs, setProcs] = useState<ProcessView[]>(initial);
  const [error, setError] = useState<string | null>(null);
  const [paused, setPaused] = useState(false);
  const [search, setSearch] = useState("");
  const [namespace, setNamespace] = useState<string>("__all__");
  const [lastTick, setLastTick] = useState<number>(Date.now());

  useEffect(() => {
    if (paused) return;
    let cancelled = false;
    let timer: ReturnType<typeof setTimeout> | null = null;
    async function tick() {
      try {
        const r = await fetch("/api/pm2/procs", { cache: "no-store" });
        const data = (await r.json()) as { procs: ProcessView[]; error?: string };
        if (cancelled) return;
        setProcs(data.procs ?? []);
        setError(data.error ?? null);
        setLastTick(Date.now());
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
  }, [paused]);

  const namespaces = useMemo(() => {
    const set = new Set<string>();
    for (const p of procs) set.add(p.namespace);
    return Array.from(set).sort();
  }, [procs]);

  const filtered = useMemo(() => {
    return procs.filter((p) => {
      if (namespace !== "__all__" && p.namespace !== namespace) return false;
      if (search) {
        const needle = search.toLowerCase();
        if (!p.name.toLowerCase().includes(needle) && !p.id.toLowerCase().includes(needle)) {
          return false;
        }
      }
      return true;
    });
  }, [procs, search, namespace]);

  return (
    <div className="space-y-4">
      <DaemonToolbar
        canWrite={canWrite}
        canDelete={canDelete}
        paused={paused}
        onPauseToggle={() => setPaused((p) => !p)}
        lastTick={lastTick}
      />

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

      <Card>
        <CardHeader className="space-y-3">
          <div className="flex items-center justify-between">
            <CardTitle className="flex items-center gap-2">
              All instances
              <Badge variant="secondary">{filtered.length}</Badge>
              {filtered.length !== procs.length ? (
                <span className="text-xs text-muted-foreground">of {procs.length}</span>
              ) : null}
            </CardTitle>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <Input
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Filter by name…"
              className="max-w-xs"
            />
            <Select value={namespace} onValueChange={setNamespace}>
              <SelectTrigger className="w-44">
                <SelectValue placeholder="All namespaces" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="__all__">All namespaces</SelectItem>
                {namespaces.map((ns) => (
                  <SelectItem key={ns} value={ns}>
                    {ns}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        </CardHeader>
        <CardContent>
          {filtered.length === 0 ? (
            <p className="text-sm text-muted-foreground">
              No processes match.{" "}
              {canWrite ? (
                <Link className="underline" href="/processes/new">
                  Add one
                </Link>
              ) : null}
            </p>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead className="border-b border-border text-left text-xs uppercase text-muted-foreground">
                  <tr>
                    <th className="py-2 pr-3">Name</th>
                    <th className="py-2 pr-3">Namespace</th>
                    <th className="py-2 pr-3">Inst.</th>
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
                  {filtered.map((p) => (
                    <tr key={p.id}>
                      <td className="py-2 pr-3 font-mono">
                        <Link
                          className="underline-offset-2 hover:underline"
                          href={`/processes/${encodeURIComponent(p.name)}`}
                        >
                          {p.name}
                        </Link>
                      </td>
                      <td className="py-2 pr-3 text-muted-foreground">{p.namespace}</td>
                      <td className="py-2 pr-3 text-muted-foreground">{p.instance_id}</td>
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
                          name={p.name}
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
          )}
        </CardContent>
      </Card>
    </div>
  );
}

function DaemonToolbar({
  canWrite,
  canDelete,
  paused,
  onPauseToggle,
  lastTick,
}: Perms & { paused: boolean; onPauseToggle: () => void; lastTick: number }) {
  const [pending, start] = useTransition();
  const [stopAllOpen, setStopAllOpen] = useState(false);
  const [deleteAllOpen, setDeleteAllOpen] = useState(false);

  const run = (label: string, fn: () => Promise<unknown>) =>
    start(async () => {
      try {
        await fn();
        toast.success(label);
      } catch (e) {
        toast.error(`${label}: ${(e as Error).message}`);
      }
    });

  return (
    <div className="flex flex-wrap items-center justify-between gap-2 rounded-lg border border-border bg-card p-3">
      <div className="flex flex-wrap items-center gap-2">
        {canWrite ? (
          <>
            <Link href="/processes/new">
              <Button size="sm">
                <Plus className="h-4 w-4" /> New
              </Button>
            </Link>
            <Link href="/processes/import">
              <Button size="sm" variant="outline">
                <Upload className="h-4 w-4" /> Import ecosystem
              </Button>
            </Link>
            <Button
              size="sm"
              variant="outline"
              disabled={pending}
              onClick={() => run("Saved dump", () => saveDumpAction())}
            >
              <Save className="h-4 w-4" /> Save dump
            </Button>
            <Button
              size="sm"
              variant="outline"
              disabled={pending}
              onClick={() => run("Started all", () => startAllAction())}
            >
              <Play className="h-4 w-4" /> Start all
            </Button>
            <Button
              size="sm"
              variant="outline"
              disabled={pending}
              onClick={() => setStopAllOpen(true)}
            >
              <Pause className="h-4 w-4" /> Stop all
            </Button>
          </>
        ) : null}
        {canDelete ? (
          <Button
            size="sm"
            variant="outline"
            className="text-red-600 dark:text-red-400"
            disabled={pending}
            onClick={() => setDeleteAllOpen(true)}
          >
            <Trash2 className="h-4 w-4" /> Delete all
          </Button>
        ) : null}
      </div>
      <div className="flex items-center gap-2 text-xs text-muted-foreground">
        <span>updated {timeAgo(lastTick)}</span>
        <Button size="sm" variant="ghost" onClick={onPauseToggle}>
          <RefreshCcw className={`h-4 w-4 ${paused ? "" : "animate-spin-slow"}`} />
          {paused ? "Resume polling" : "Pause polling"}
        </Button>
      </div>

      {canWrite ? (
        <ConfirmDeleteDialog
          open={stopAllOpen}
          onOpenChange={setStopAllOpen}
          name="all processes"
          title="Stop all processes?"
          description="SIGTERMs every managed instance, then SIGKILLs anything that doesn't exit within its kill_timeout. The dump is preserved, so a subsequent `Start all` brings them back."
          challenge="stop all"
          confirmLabel="Stop all"
          action={() => stopAllAction()}
          successMessage="Stopped all"
        />
      ) : null}
      {canDelete ? (
        <ConfirmDeleteDialog
          open={deleteAllOpen}
          onOpenChange={setDeleteAllOpen}
          name="all processes"
          title="Delete all processes?"
          description="Stops every instance and removes every spec from the dump.json. Log files on disk stay, but the daemon will no longer supervise anything until you re-add apps."
          challenge="delete all"
          confirmLabel="Delete all"
          action={() => deleteAllAction()}
          successMessage="Deleted all"
        />
      ) : null}
    </div>
  );
}

function timeAgo(ts: number): string {
  const s = Math.max(0, Math.round((Date.now() - ts) / 1000));
  if (s < 60) return `${s}s ago`;
  return `${Math.floor(s / 60)}m ago`;
}
