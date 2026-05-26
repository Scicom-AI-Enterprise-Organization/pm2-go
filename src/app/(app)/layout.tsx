import { requireUser } from "@/lib/rbac";
import { Sidebar } from "@/components/nav/sidebar";
import { Topbar } from "@/components/nav/topbar";
import { SidebarStateProvider } from "@/components/nav/sidebar-state";

export default async function AppLayout({ children }: { children: React.ReactNode }) {
  const user = await requireUser();
  const canSeeAdmin = user.permissions.some((p) =>
    ["users:read", "roles:read", "invites:read"].includes(p),
  );
  const canSeeProcesses = user.permissions.includes("processes:read");

  return (
    <SidebarStateProvider>
      <div className="flex h-screen bg-background">
        <Sidebar canSeeProcesses={canSeeProcesses} canSeeAdmin={canSeeAdmin} />
        <div className="flex min-w-0 flex-1 flex-col overflow-hidden">
          <Topbar />
          <main className="min-w-0 flex-1 overflow-auto p-6">{children}</main>
        </div>
      </div>
    </SidebarStateProvider>
  );
}
