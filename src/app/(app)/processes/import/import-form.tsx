"use client";

import { useState, useTransition } from "react";
import { useRouter } from "next/navigation";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { importEcosystemAction } from "../actions";

const SAMPLE = JSON.stringify(
  {
    apps: [
      {
        name: "api",
        script: "/srv/api/index.js",
        interpreter: "node",
        instances: 4,
        namespace: "prod",
        env: { NODE_ENV: "production" },
        watch: ["/srv/api/src"],
        max_memory_restart: "512M",
      },
    ],
  },
  null,
  2,
);

export function ImportForm() {
  const router = useRouter();
  const [text, setText] = useState(SAMPLE);
  const [preview, setPreview] = useState<string[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [pending, start] = useTransition();

  function previewParse() {
    setError(null);
    try {
      const data = JSON.parse(text) as { apps?: Array<{ name?: string }> } | Array<{ name?: string }>;
      const apps = Array.isArray(data) ? data : data.apps ?? [];
      if (!Array.isArray(apps) || apps.length === 0) {
        setError("expected `apps: [...]` with at least one entry");
        return;
      }
      const missing = apps.find((a) => !a?.name);
      if (missing) {
        setError("every app needs a `name`");
        return;
      }
      setPreview(apps.map((a) => a.name as string));
    } catch (e) {
      setError(`invalid JSON: ${(e as Error).message}`);
    }
  }

  function onSubmit() {
    start(async () => {
      try {
        const res = await importEcosystemAction(text);
        toast.success(`Imported ${res.applied.length} app(s): ${res.applied.join(", ")}`);
        router.push("/processes");
      } catch (e) {
        setError((e as Error).message);
        toast.error(`Import failed: ${(e as Error).message}`);
      }
    });
  }

  return (
    <div className="space-y-3">
      <Textarea
        value={text}
        onChange={(e) => {
          setText(e.target.value);
          setPreview(null);
        }}
        rows={18}
        className="font-mono text-xs"
        spellCheck={false}
      />
      {error ? (
        <p className="rounded-md bg-destructive/10 p-2 text-sm text-destructive">{error}</p>
      ) : null}
      {preview ? (
        <p className="rounded-md bg-primary/10 p-2 text-sm">
          Ready to apply: <span className="font-mono">{preview.join(", ")}</span>
        </p>
      ) : null}
      <div className="flex justify-end gap-2">
        <Button type="button" variant="outline" onClick={previewParse} disabled={pending}>
          Validate
        </Button>
        <Button type="button" onClick={onSubmit} disabled={pending}>
          {pending ? "Applying…" : "Apply"}
        </Button>
      </div>
    </div>
  );
}
