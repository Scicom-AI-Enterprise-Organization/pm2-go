"use client";

import { useTransition } from "react";
import { toast } from "sonner";
import { useRouter } from "next/navigation";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { Switch } from "@/components/ui/switch";
import type { ProcessSpec } from "@/lib/pm2";
import { startAction, upsertAction } from "./actions";

type Mode = "create" | "edit";

type Props = {
  mode: Mode;
  initial?: ProcessSpec;
};

/**
 * Reusable spec form used by both `/processes/new` (create) and
 * `/processes/[name]/edit`. In edit mode the name is read-only because it's
 * the spec's primary key on the daemon.
 */
export function ProcessForm({ mode, initial }: Props) {
  const router = useRouter();
  const [pending, start] = useTransition();
  const isEdit = mode === "edit";

  function onSubmit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const fd = new FormData(e.currentTarget);
    const env: Record<string, string> = {};
    for (const line of String(fd.get("env") ?? "").split("\n")) {
      const eq = line.indexOf("=");
      if (eq > 0) env[line.slice(0, eq).trim()] = line.slice(eq + 1).trim();
    }
    const watch = splitCSV(fd.get("watch"));
    const ignoreWatch = splitCSV(fd.get("ignore_watch"));
    const argList = String(fd.get("args") ?? "")
      .split(/\s+/)
      .filter(Boolean);

    const name = String(fd.get("name") ?? "").trim();
    const maxMem = parseMemory(String(fd.get("max_memory_restart") ?? "").trim());

    const spec: Partial<ProcessSpec> = {
      name,
      script: String(fd.get("script") ?? "").trim(),
      interpreter: emptyToUndefined(String(fd.get("interpreter") ?? "")),
      cwd: emptyToUndefined(String(fd.get("cwd") ?? "")),
      args: argList.length ? argList : undefined,
      instances: Number(fd.get("instances") || 1),
      namespace: String(fd.get("namespace") ?? "").trim() || "default",
      env: Object.keys(env).length ? env : undefined,
      watch: watch.length ? watch : undefined,
      ignore_watch: ignoreWatch.length ? ignoreWatch : undefined,
      max_restarts: Number(fd.get("max_restarts") || 16),
      kill_timeout: Number(fd.get("kill_timeout_ms") || 1600) * 1_000_000,
      autorestart_disabled: fd.get("autorestart") === "off" ? true : false,
      max_memory_restart: maxMem,
      log_max_size_mb: Number(fd.get("log_max_size_mb") || 0) || undefined,
      log_max_backups: Number(fd.get("log_max_backups") || 0) || undefined,
      log_max_age_days: Number(fd.get("log_max_age_days") || 0) || undefined,
    };

    start(async () => {
      try {
        if (isEdit) {
          await upsertAction(spec);
          toast.success(`Saved ${name}`);
          router.push(`/processes/${encodeURIComponent(name)}`);
        } else {
          await startAction(spec);
          toast.success(`Started ${name}`);
        }
      } catch (err) {
        toast.error(`${isEdit ? "Save" : "Start"} ${name} failed: ${(err as Error).message}`);
      }
    });
  }

  return (
    <form onSubmit={onSubmit} className="space-y-4">
      <div className="grid gap-4 sm:grid-cols-2">
        <Field
          label="Name"
          name="name"
          required
          placeholder="my-api"
          defaultValue={initial?.name}
          readOnly={isEdit}
        />
        <Field
          label="Namespace"
          name="namespace"
          placeholder="default"
          defaultValue={initial?.namespace}
        />
        <Field
          label="Script"
          name="script"
          required
          placeholder="/srv/app/index.js"
          defaultValue={initial?.script}
        />
        <Field
          label="Interpreter"
          name="interpreter"
          placeholder="node, python3, /bin/bash"
          defaultValue={initial?.interpreter}
        />
        <Field
          label="Working dir"
          name="cwd"
          placeholder="/srv/app"
          defaultValue={initial?.cwd}
        />
        <Field
          label="Args"
          name="args"
          placeholder="--port 3000 --verbose"
          defaultValue={initial?.args?.join(" ")}
        />
        <Field
          label="Instances"
          name="instances"
          type="number"
          defaultValue={initial?.instances ?? 1}
          min={1}
        />
        <Field
          label="Max restarts"
          name="max_restarts"
          type="number"
          defaultValue={initial?.max_restarts ?? 16}
          min={0}
        />
        <Field
          label="Kill timeout (ms)"
          name="kill_timeout_ms"
          type="number"
          defaultValue={initial?.kill_timeout ? Math.round(initial.kill_timeout / 1_000_000) : 1600}
          min={0}
        />
        <Field
          label="Max memory restart"
          name="max_memory_restart"
          placeholder="e.g. 300M, 1G"
          defaultValue={initial?.max_memory_restart ? `${initial.max_memory_restart}` : ""}
        />
      </div>

      <Field
        label="Watch (comma-separated paths)"
        name="watch"
        placeholder="src, lib"
        defaultValue={initial?.watch?.join(", ")}
      />
      <Field
        label="Ignore watch (globs)"
        name="ignore_watch"
        placeholder="*.log, *.tmp"
        defaultValue={initial?.ignore_watch?.join(", ")}
      />

      <div className="grid gap-4 sm:grid-cols-3">
        <Field
          label="Log rotate size (MB)"
          name="log_max_size_mb"
          type="number"
          defaultValue={initial?.log_max_size_mb ?? 0}
          min={0}
        />
        <Field
          label="Log backups"
          name="log_max_backups"
          type="number"
          defaultValue={initial?.log_max_backups ?? 0}
          min={0}
        />
        <Field
          label="Log max age (days)"
          name="log_max_age_days"
          type="number"
          defaultValue={initial?.log_max_age_days ?? 0}
          min={0}
        />
      </div>

      <div className="flex items-center gap-3">
        <Switch
          id="autorestart"
          name="autorestart"
          defaultChecked={initial ? !initial.autorestart_disabled : true}
          value="on"
        />
        <Label htmlFor="autorestart">Autorestart on exit</Label>
      </div>

      <div>
        <Label htmlFor="env">Environment (KEY=VALUE per line)</Label>
        <Textarea
          id="env"
          name="env"
          rows={4}
          placeholder="NODE_ENV=production&#10;PORT=3000"
          defaultValue={
            initial?.env
              ? Object.entries(initial.env)
                  .map(([k, v]) => `${k}=${v}`)
                  .join("\n")
              : ""
          }
        />
      </div>

      <div className="flex justify-end gap-2">
        <Button type="submit" disabled={pending}>
          {pending ? (isEdit ? "Saving…" : "Starting…") : isEdit ? "Save changes" : "Start process"}
        </Button>
      </div>
    </form>
  );
}

function Field({
  label,
  name,
  required,
  type = "text",
  defaultValue,
  min,
  placeholder,
  readOnly,
}: {
  label: string;
  name: string;
  required?: boolean;
  type?: string;
  defaultValue?: string | number;
  min?: number;
  placeholder?: string;
  readOnly?: boolean;
}) {
  return (
    <div>
      <Label htmlFor={name}>{label}</Label>
      <Input
        id={name}
        name={name}
        type={type}
        required={required}
        defaultValue={defaultValue}
        min={min}
        placeholder={placeholder}
        readOnly={readOnly}
        className={readOnly ? "bg-muted text-muted-foreground" : undefined}
      />
    </div>
  );
}

function splitCSV(v: FormDataEntryValue | null): string[] {
  return String(v ?? "")
    .split(",")
    .map((s) => s.trim())
    .filter(Boolean);
}

function emptyToUndefined(s: string): string | undefined {
  const t = s.trim();
  return t === "" ? undefined : t;
}

function parseMemory(s: string): number | undefined {
  if (!s) return undefined;
  const m = /^(\d+(?:\.\d+)?)\s*([kKmMgG]?)/.exec(s.trim());
  if (!m) return undefined;
  const n = parseFloat(m[1]);
  const suffix = m[2].toUpperCase();
  const mult: Record<string, number> = { K: 1024, M: 1024 ** 2, G: 1024 ** 3 };
  return Math.round(n * (mult[suffix] ?? 1));
}
