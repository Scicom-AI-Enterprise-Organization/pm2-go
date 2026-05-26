import { requirePermission, hasPermission, getCurrentUser } from "@/lib/rbac";
import { ApiDocsView } from "./api-docs-view";

export const dynamic = "force-dynamic";

export default async function ApiDocsPage() {
  await requirePermission("processes:read");
  const me = await getCurrentUser();
  const canSeeToken = hasPermission(me, "processes:write");

  // Read the daemon URL + token server-side. Token only flows to the client
  // for users who can already mutate processes (processes:write).
  const baseURL = process.env.PM2_GO_DAEMON_URL ?? "http://127.0.0.1:9615";
  let token: string | null = null;
  if (canSeeToken) {
    token = process.env.PM2_GO_DAEMON_TOKEN ?? null;
    if (!token) {
      try {
        const { readFileSync } = await import("node:fs");
        const { homedir } = await import("node:os");
        const { join } = await import("node:path");
        const p = process.env.PM2_GO_HOME
          ? join(process.env.PM2_GO_HOME, "api-token")
          : join(homedir(), ".pm2-go", "api-token");
        token = readFileSync(p, "utf8").trim();
      } catch {
        token = null;
      }
    }
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Daemon API</h1>
        <p className="mt-1 text-sm text-muted-foreground">
          REST + ndjson surface on the pm2-go daemon. Use it from CI, scripts, or
          other backends.
        </p>
      </div>

      <ApiDocsView baseURL={baseURL} token={token} />
    </div>
  );
}
