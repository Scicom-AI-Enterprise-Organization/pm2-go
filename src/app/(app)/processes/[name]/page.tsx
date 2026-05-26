import { notFound } from "next/navigation";
import { requirePermission, hasPermission, getCurrentUser } from "@/lib/rbac";
import { describeApp } from "@/lib/pm2";
import { DetailView } from "./detail-view";

export const dynamic = "force-dynamic";

export default async function ProcessDetailPage({
  params,
}: {
  params: Promise<{ name: string }>;
}) {
  await requirePermission("processes:read");
  const me = await getCurrentUser();
  const canLogs = hasPermission(me, "processes:logs");
  const canWrite = hasPermission(me, "processes:write");
  const canDelete = hasPermission(me, "processes:delete");
  const { name: rawName } = await params;
  const name = decodeURIComponent(rawName);

  let initial;
  try {
    initial = await describeApp(name);
  } catch (e) {
    if ((e as Error).message.includes("404")) notFound();
    throw e;
  }

  return (
    <div>
      <DetailView
        name={name}
        initial={initial}
        canWrite={canWrite}
        canDelete={canDelete}
        canLogs={canLogs}
      />
    </div>
  );
}
