"use client";

import { useEffect, useRef, useState } from "react";

type Line = { runtime: string; stream: string; text: string; time?: number };

export function LogStream({ name, initial }: { name: string; initial: Line[] }) {
  const [lines, setLines] = useState<Line[]>(initial);
  const [paused, setPaused] = useState(false);
  const bufferRef = useRef<Line[]>([]);
  const scrollRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const ctrl = new AbortController();
    (async () => {
      try {
        const res = await fetch(`/api/pm2/logs/stream?name=${encodeURIComponent(name)}`, {
          signal: ctrl.signal,
        });
        if (!res.ok || !res.body) return;
        const reader = res.body.getReader();
        const decoder = new TextDecoder();
        let buf = "";
        for (;;) {
          const { value, done } = await reader.read();
          if (done) break;
          buf += decoder.decode(value, { stream: true });
          let nl;
          while ((nl = buf.indexOf("\n")) >= 0) {
            const raw = buf.slice(0, nl);
            buf = buf.slice(nl + 1);
            if (!raw.trim()) continue;
            try {
              const l = JSON.parse(raw) as Line;
              if (paused) {
                bufferRef.current.push(l);
              } else {
                setLines((prev) => trim([...prev, l]));
              }
            } catch {
              /* ignore */
            }
          }
        }
      } catch {
        /* aborted */
      }
    })();
    return () => ctrl.abort();
  }, [name, paused]);

  // Flush buffer when unpaused.
  useEffect(() => {
    if (!paused && bufferRef.current.length > 0) {
      setLines((prev) => trim([...prev, ...bufferRef.current]));
      bufferRef.current = [];
    }
  }, [paused]);

  // Auto-scroll to bottom on new lines unless user scrolled up.
  useEffect(() => {
    const el = scrollRef.current;
    if (!el) return;
    const nearBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 60;
    if (nearBottom) el.scrollTop = el.scrollHeight;
  }, [lines]);

  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between">
        <p className="text-xs text-muted-foreground">{lines.length} line(s)</p>
        <div className="flex gap-2">
          <button
            type="button"
            className="rounded border border-border px-2 py-1 text-xs hover:bg-muted"
            onClick={() => setPaused((p) => !p)}
          >
            {paused ? "Resume" : "Pause"}
          </button>
          <button
            type="button"
            className="rounded border border-border px-2 py-1 text-xs hover:bg-muted"
            onClick={() => setLines([])}
          >
            Clear
          </button>
        </div>
      </div>
      <div
        ref={scrollRef}
        className="h-[60vh] overflow-y-auto rounded bg-zinc-950 p-3 font-mono text-xs leading-relaxed text-zinc-100"
      >
        {lines.map((l, i) => (
          <div key={i} className={l.stream === "err" ? "text-red-300" : "text-zinc-200"}>
            <span className="select-none text-zinc-500">[{l.runtime}/{l.stream}]</span>{" "}
            {l.text.replace(/\n+$/, "")}
          </div>
        ))}
      </div>
    </div>
  );
}

function trim(arr: Line[]): Line[] {
  const max = 5000;
  return arr.length > max ? arr.slice(arr.length - max) : arr;
}
