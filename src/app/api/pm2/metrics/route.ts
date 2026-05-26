import { NextRequest } from "next/server";
import { requirePermission } from "@/lib/rbac";
import { metricsHistory } from "@/lib/pm2";

export const dynamic = "force-dynamic";

export async function GET(req: NextRequest) {
  await requirePermission("processes:read");
  const name = req.nextUrl.searchParams.get("name");
  if (!name) return new Response("missing name", { status: 400 });
  try {
    const series = await metricsHistory(name);
    return Response.json({ series });
  } catch (e) {
    return Response.json({ series: {}, error: (e as Error).message }, { status: 502 });
  }
}
