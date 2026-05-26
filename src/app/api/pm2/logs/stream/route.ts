/**
 * Proxies ndjson log streams from the pm2-go daemon to the browser. We keep the
 * daemon token server-side and forward the upstream stream untouched.
 */

import { NextRequest } from "next/server";
import { requirePermission } from "@/lib/rbac";
import { streamLogs } from "@/lib/pm2";

export const dynamic = "force-dynamic";

export async function GET(req: NextRequest) {
  await requirePermission("processes:logs");
  const name = req.nextUrl.searchParams.get("name");
  if (!name) return new Response("missing name", { status: 400 });

  const upstream = await streamLogs(name, req.signal).catch((e: Error) => e);
  if (upstream instanceof Error) {
    return new Response(upstream.message, { status: 502 });
  }
  return new Response(upstream.body, {
    status: 200,
    headers: {
      "Content-Type": "application/x-ndjson",
      "Cache-Control": "no-cache, no-transform",
      "X-Accel-Buffering": "no",
    },
  });
}
