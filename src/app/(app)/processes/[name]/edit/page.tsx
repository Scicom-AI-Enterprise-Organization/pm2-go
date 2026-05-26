import Link from "next/link";
import { notFound } from "next/navigation";
import { requirePermission } from "@/lib/rbac";
import { describeApp } from "@/lib/pm2";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { ArrowLeft } from "lucide-react";
import { ProcessForm } from "../../process-form";

export const dynamic = "force-dynamic";

export default async function EditProcessPage({
  params,
}: {
  params: Promise<{ name: string }>;
}) {
  await requirePermission("processes:write");
  const { name: rawName } = await params;
  const name = decodeURIComponent(rawName);

  let data;
  try {
    data = await describeApp(name);
  } catch (e) {
    if ((e as Error).message.includes("404")) notFound();
    throw e;
  }

  return (
    <div className="mx-auto max-w-3xl space-y-4">
      <div>
        <Link
          href={`/processes/${encodeURIComponent(name)}`}
          className="inline-flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground"
        >
          <ArrowLeft className="h-3 w-3" /> Back to {name}
        </Link>
        <h1 className="mt-1 text-3xl font-bold tracking-tight">Edit {name}</h1>
        <p className="mt-1 text-sm text-muted-foreground">
          Saving stops and replaces the existing app on the daemon. Logs are preserved.
        </p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Configuration</CardTitle>
        </CardHeader>
        <CardContent>
          <ProcessForm mode="edit" initial={data.spec} />
        </CardContent>
      </Card>
    </div>
  );
}
