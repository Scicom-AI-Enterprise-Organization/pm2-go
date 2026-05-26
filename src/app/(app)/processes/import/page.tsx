import Link from "next/link";
import { requirePermission } from "@/lib/rbac";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { ArrowLeft } from "lucide-react";
import { ImportForm } from "./import-form";

export const dynamic = "force-dynamic";

export default async function ImportEcosystemPage() {
  await requirePermission("processes:write");
  return (
    <div className="space-y-4">
      <div>
        <Link
          href="/processes"
          className="inline-flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground"
        >
          <ArrowLeft className="h-3 w-3" /> All processes
        </Link>
        <h1 className="mt-1 text-3xl font-bold tracking-tight">Import ecosystem file</h1>
        <p className="mt-1 text-sm text-muted-foreground">
          Paste an{" "}
          <code className="font-mono">ecosystem.config.json</code> payload — each{" "}
          <code className="font-mono">apps[]</code> entry is sent to the daemon as
          add-or-replace.
        </p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Ecosystem JSON</CardTitle>
        </CardHeader>
        <CardContent>
          <ImportForm />
        </CardContent>
      </Card>
    </div>
  );
}
