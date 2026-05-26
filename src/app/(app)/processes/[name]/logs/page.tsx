import Link from "next/link";
import { requirePermission } from "@/lib/rbac";
import { tailLogs } from "@/lib/pm2";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { ArrowLeft } from "lucide-react";
import { LogStream } from "./log-stream";

export const dynamic = "force-dynamic";

export default async function ProcessLogsPage({
  params,
}: {
  params: Promise<{ name: string }>;
}) {
  await requirePermission("processes:logs");
  const { name: rawName } = await params;
  const name = decodeURIComponent(rawName);

  // Seed with recent history so the page isn't empty on first render.
  const history = await tailLogs(name, 100).catch(() => ({} as Record<string, string[]>));
  const initialLines: { runtime: string; stream: string; text: string }[] = [];
  for (const [key, lines] of Object.entries(history)) {
    const [runtime, stream] = key.split(":");
    for (const text of lines) initialLines.push({ runtime, stream, text });
  }

  return (
    <div className="mx-auto max-w-6xl space-y-4">
      <div className="flex items-start justify-between gap-4">
        <div>
          <Link
            href={`/processes/${encodeURIComponent(name)}`}
            className="inline-flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground"
          >
            <ArrowLeft className="h-3 w-3" /> Back
          </Link>
          <h1 className="mt-1 text-3xl font-bold tracking-tight">{name} — logs</h1>
          <p className="text-sm text-muted-foreground">
            Live stream of stdout (white) and stderr (red). History is the most recent 100 lines.
          </p>
        </div>
        <Link href={`/processes/${encodeURIComponent(name)}`}>
          <Button variant="outline">Details</Button>
        </Link>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Stream</CardTitle>
        </CardHeader>
        <CardContent>
          <LogStream name={name} initial={initialLines} />
        </CardContent>
      </Card>
    </div>
  );
}
