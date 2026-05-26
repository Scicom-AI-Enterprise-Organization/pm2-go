import Link from "next/link";
import { requirePermission } from "@/lib/rbac";
import { ArrowLeft } from "lucide-react";
import { NewProcessForm } from "./new-process-form";

export const dynamic = "force-dynamic";

export default async function NewProcessPage() {
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
        <h1 className="mt-1 text-3xl font-bold tracking-tight">Start a new process</h1>
        <p className="mt-1 text-sm text-muted-foreground">
          The daemon spawns and supervises the script. State is persisted to{" "}
          <code className="font-mono">~/.pm2-go/dump.json</code>.
        </p>
      </div>

      <NewProcessForm />
    </div>
  );
}
