import { requirePermission, hasPermission, getCurrentUser } from "@/lib/rbac";
import { listProcs } from "@/lib/pm2";
import { ProcessesView } from "./processes-view";

export const dynamic = "force-dynamic";
export const revalidate = 0;

export default async function ProcessesPage() {
  await requirePermission("processes:read");
  const me = await getCurrentUser();
  const canWrite = hasPermission(me, "processes:write");
  const canDelete = hasPermission(me, "processes:delete");

  let initial;
  try {
    initial = await listProcs();
  } catch {
    initial = [];
  }

  return (
    <div className="mx-auto max-w-6xl space-y-6">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Processes</h1>
        <p className="mt-2 text-muted-foreground">
          pm2-go managed processes — auto-refreshing every 2s.
        </p>
      </div>
      <ProcessesView initial={initial} canWrite={canWrite} canDelete={canDelete} />
    </div>
  );
}
