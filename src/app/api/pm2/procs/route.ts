/**
 * Read-only proxy for the daemon's `/v1/procs` endpoint. Used by the list
 * page's polling client component so the daemon token stays server-side.
 */

import { requirePermission } from "@/lib/rbac";
import { listProcs } from "@/lib/pm2";

export const dynamic = "force-dynamic";

export async function GET() {
  await requirePermission("processes:read");
  try {
    const procs = await listProcs();
    return Response.json({ procs });
  } catch (e) {
    return Response.json({ procs: [], error: (e as Error).message }, { status: 502 });
  }
}
