"use client";

import { useEffect, useState } from "react";
import type { MetricSample } from "@/lib/pm2";

type Series = Record<string, MetricSample[]>;

const POLL_MS = 2000;

export function MetricsChart({ name }: { name: string }) {
  const [series, setSeries] = useState<Series>({});
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    let timer: ReturnType<typeof setTimeout> | null = null;
    async function tick() {
      try {
        const r = await fetch(`/api/pm2/metrics?name=${encodeURIComponent(name)}`, {
          cache: "no-store",
        });
        const data = (await r.json()) as { series?: Series; error?: string };
        if (cancelled) return;
        setSeries(data.series ?? {});
        setError(data.error ?? null);
      } catch (e) {
        if (!cancelled) setError((e as Error).message);
      } finally {
        if (!cancelled) timer = setTimeout(tick, POLL_MS);
      }
    }
    tick();
    return () => {
      cancelled = true;
      if (timer) clearTimeout(timer);
    };
  }, [name]);

  const runtimeIds = Object.keys(series).sort();

  if (runtimeIds.length === 0) {
    return (
      <p className="text-sm text-muted-foreground">
        {error ?? "Waiting for samples… metrics arrive after the first second of runtime."}
      </p>
    );
  }

  return (
    <div className="space-y-4">
      {runtimeIds.map((id) => (
        <RuntimeSparklines key={id} runtimeId={id} samples={series[id]} />
      ))}
    </div>
  );
}

function RuntimeSparklines({
  runtimeId,
  samples,
}: {
  runtimeId: string;
  samples: MetricSample[];
}) {
  if (samples.length < 2) {
    return (
      <div className="rounded-md border border-border p-3">
        <p className="text-sm font-medium">{runtimeId}</p>
        <p className="mt-1 text-xs text-muted-foreground">Collecting samples…</p>
      </div>
    );
  }
  const cpus = samples.map((s) => s.cpu);
  const mems = samples.map((s) => s.mem);
  const lastCPU = cpus[cpus.length - 1];
  const lastMem = mems[mems.length - 1];
  const peakCPU = Math.max(...cpus);
  const peakMem = Math.max(...mems);

  return (
    <div className="rounded-md border border-border p-3">
      <div className="mb-2 flex items-center justify-between">
        <p className="text-sm font-medium">{runtimeId}</p>
        <p className="text-xs text-muted-foreground">{samples.length} samples (~1s each)</p>
      </div>
      <div className="grid gap-3 sm:grid-cols-2">
        <ChartPanel
          label="CPU"
          unit="%"
          color="oklch(0.6231 0.1880 259.8145)"
          values={cpus}
          last={lastCPU}
          peak={peakCPU}
          format={(v) => `${v.toFixed(1)}%`}
        />
        <ChartPanel
          label="Memory"
          unit="bytes"
          color="oklch(0.5461 0.2152 262.8809)"
          values={mems}
          last={lastMem}
          peak={peakMem}
          format={(v) => humanBytes(v)}
        />
      </div>
    </div>
  );
}

function ChartPanel({
  label,
  color,
  values,
  last,
  peak,
  format,
}: {
  label: string;
  unit: string;
  color: string;
  values: number[];
  last: number;
  peak: number;
  format: (n: number) => string;
}) {
  const max = peak === 0 ? 1 : peak;
  const points = values
    .map((v, i) => {
      const x = (i / Math.max(1, values.length - 1)) * 100;
      const y = 100 - (v / max) * 100;
      return `${x.toFixed(2)},${y.toFixed(2)}`;
    })
    .join(" ");
  const area = `0,100 ${points} 100,100`;
  return (
    <div>
      <div className="mb-1 flex items-end justify-between text-xs text-muted-foreground">
        <span className="uppercase tracking-wide">{label}</span>
        <span className="font-mono text-foreground">{format(last)}</span>
      </div>
      <svg viewBox="0 0 100 100" preserveAspectRatio="none" className="h-16 w-full">
        <polygon points={area} fill={color} fillOpacity={0.15} />
        <polyline points={points} fill="none" stroke={color} strokeWidth={1.2} />
      </svg>
      <p className="mt-1 text-[10px] text-muted-foreground">peak {format(peak)}</p>
    </div>
  );
}

function humanBytes(n: number): string {
  const k = 1024;
  if (n < k) return `${n} B`;
  if (n < k * k) return `${(n / k).toFixed(1)} KB`;
  if (n < k * k * k) return `${(n / (k * k)).toFixed(1)} MB`;
  return `${(n / (k * k * k)).toFixed(2)} GB`;
}
