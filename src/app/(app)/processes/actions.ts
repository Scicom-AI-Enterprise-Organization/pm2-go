"use server";

import { revalidatePath } from "next/cache";
import { requirePermission } from "@/lib/rbac";
import {
  deleteAll,
  deleteApp,
  reloadApp,
  restartApp,
  saveDump,
  startAll,
  startApp,
  stopAll,
  stopApp,
  type ProcessSpec,
} from "@/lib/pm2";

/**
 * IMPORTANT: actions called from `useTransition(() => action())` must not call
 * `next/navigation.redirect()` — it throws a `NEXT_REDIRECT` error that the
 * client transition surfaces as a normal exception (you'll see "Delete failed:
 * NEXT_REDIRECT" in a toast). Navigation happens client-side via the caller's
 * router.push/redirect on success. Actions invoked via `<form action={…}>` are
 * fine because Next intercepts redirect()s in that path.
 */

export async function startAction(spec: Partial<ProcessSpec>) {
  await requirePermission("processes:write");
  await startApp(spec);
  revalidatePath("/processes");
  if (spec.name) revalidatePath(`/processes/${spec.name}`);
}

export async function stopAction(name: string) {
  await requirePermission("processes:write");
  await stopApp(name);
  revalidatePath("/processes");
  revalidatePath(`/processes/${name}`);
}

export async function restartAction(name: string) {
  await requirePermission("processes:write");
  await restartApp(name);
  revalidatePath("/processes");
  revalidatePath(`/processes/${name}`);
}

export async function reloadAction(name: string) {
  await requirePermission("processes:write");
  await reloadApp(name);
  revalidatePath(`/processes/${name}`);
}

export async function deleteAction(name: string) {
  await requirePermission("processes:delete");
  await deleteApp(name);
  revalidatePath("/processes");
}

/**
 * Replace an existing spec without redirecting. Used by the edit page so the
 * user stays on the form for further changes if they want.
 */
export async function upsertAction(spec: Partial<ProcessSpec>) {
  await requirePermission("processes:write");
  await startApp(spec);
  revalidatePath("/processes");
  if (spec.name) revalidatePath(`/processes/${spec.name}`);
}

export async function saveDumpAction() {
  await requirePermission("processes:write");
  await saveDump();
}

export async function startAllAction() {
  await requirePermission("processes:write");
  await startAll();
  revalidatePath("/processes");
}

export async function stopAllAction() {
  await requirePermission("processes:write");
  await stopAll();
  revalidatePath("/processes");
}

export async function deleteAllAction() {
  await requirePermission("processes:delete");
  await deleteAll();
  revalidatePath("/processes");
}

/**
 * Apply an ecosystem.config.json payload. Each `apps[]` entry becomes one
 * startApp call (add-or-replace).
 */
export async function importEcosystemAction(payload: string): Promise<{
  applied: string[];
}> {
  await requirePermission("processes:write");
  let parsed: unknown;
  try {
    parsed = JSON.parse(payload);
  } catch (e) {
    throw new Error(`invalid JSON: ${(e as Error).message}`);
  }
  const wrapper = parsed as { apps?: Partial<ProcessSpec>[] } | Partial<ProcessSpec>[];
  const apps = Array.isArray(wrapper) ? wrapper : (wrapper.apps ?? []);
  if (!Array.isArray(apps) || apps.length === 0) {
    throw new Error("expected `apps: [...]` with at least one entry");
  }
  const applied: string[] = [];
  for (const a of apps) {
    if (!a.name || !a.script) {
      throw new Error("each app needs both name and script");
    }
    await startApp(a);
    applied.push(a.name);
  }
  revalidatePath("/processes");
  return { applied };
}
