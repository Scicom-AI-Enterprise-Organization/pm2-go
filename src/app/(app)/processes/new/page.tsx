import Link from "next/link";
import { requirePermission } from "@/lib/rbac";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { ArrowLeft } from "lucide-react";
import { NewProcessForm } from "./new-process-form";

export const dynamic = "force-dynamic";

export default async function NewProcessPage() {
  await requirePermission("processes:write");
  return (
    <div className="mx-auto max-w-3xl space-y-4">
      <div>
        <Link
          href="/processes"
          className="inline-flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground"
        >
          <ArrowLeft className="h-3 w-3" /> All processes
        </Link>
        <h1 className="mt-1 text-3xl font-bold tracking-tight">Start a new process</h1>
        <p className="mt-1 text-sm text-muted-foreground">
          The daemon spawns and supervises the script. State is persisted to{" "}
          <code className="font-mono">~/.pm2-go/dump.json</code>.
        </p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Configuration</CardTitle>
        </CardHeader>
        <CardContent>
          <NewProcessForm />
          <div className="mt-4 flex justify-end gap-2">
            <Link href="/processes">
              <Button variant="ghost" type="button">
                Cancel
              </Button>
            </Link>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
