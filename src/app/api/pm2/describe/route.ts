import { NextRequest } from "next/server";
import { requirePermission } from "@/lib/rbac";
import { describeApp } from "@/lib/pm2";

export const dynamic = "force-dynamic";

export async function GET(req: NextRequest) {
  await requirePermission("processes:read");
  const name = req.nextUrl.searchParams.get("name");
  if (!name) return new Response("missing name", { status: 400 });
  try {
    const data = await describeApp(name);
    return Response.json(data);
  } catch (e) {
    const msg = (e as Error).message;
    const status = msg.includes("404") ? 404 : 502;
    return Response.json({ error: msg }, { status });
  }
}
