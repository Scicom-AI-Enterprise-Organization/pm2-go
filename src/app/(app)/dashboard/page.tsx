import { requireUser, hasPermission } from "@/lib/rbac";
import { listProcs, type ProcessView } from "@/lib/pm2";
import { DashboardView, AccountOnlyDashboard } from "./dashboard-view";

export const dynamic = "force-dynamic";
export const revalidate = 0;

export default async function DashboardPage() {
  const user = await requireUser();
  const canReadProcs = hasPermission(user, "processes:read");
  const canWrite = hasPermission(user, "processes:write");

  let initial: ProcessView[] = [];
  let daemonReachable = true;
  let daemonError: string | undefined;
  if (canReadProcs) {
    try {
      initial = await listProcs();
    } catch (e) {
      daemonReachable = false;
      daemonError = (e as Error).message;
    }
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">
          Welcome{user.name ? `, ${user.name}` : ""}
        </h1>
        <p className="mt-1 text-sm text-muted-foreground">
          {canReadProcs
            ? "Live fleet overview — auto-refreshes every 5 seconds."
            : "Your account at a glance."}
        </p>
      </div>

      {canReadProcs ? (
        <DashboardView
          initial={initial}
          canWrite={canWrite}
          daemonReachable={daemonReachable}
          daemonError={daemonError}
        />
      ) : (
        <AccountOnlyDashboard
          roles={user.roles}
          permissions={user.permissions}
          email={user.email}
          name={user.name}
        />
      )}
    </div>
  );
}
