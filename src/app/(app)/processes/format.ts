/**
 * Small display helpers shared by the processes pages.
 */

export function humanBytes(n: number): string {
  if (!n) return "—";
  const k = 1024;
  if (n < k) return `${n} B`;
  if (n < k * k) return `${(n / k).toFixed(1)} KB`;
  if (n < k * k * k) return `${(n / (k * k)).toFixed(1)} MB`;
  return `${(n / (k * k * k)).toFixed(2)} GB`;
}

export function humanDuration(seconds: number): string {
  if (!seconds) return "—";
  if (seconds < 60) return `${seconds}s`;
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m`;
  if (seconds < 86400) return `${Math.floor(seconds / 3600)}h`;
  return `${Math.floor(seconds / 86400)}d`;
}

export function stateColor(state: string): string {
  switch (state) {
    case "online":
      return "bg-emerald-500/10 text-emerald-700 dark:text-emerald-300";
    case "launching":
    case "waiting_restart":
    case "online_restarting":
      return "bg-amber-500/10 text-amber-700 dark:text-amber-300";
    case "stopping":
      return "bg-orange-500/10 text-orange-700 dark:text-orange-300";
    case "stopped":
      return "bg-zinc-500/10 text-zinc-700 dark:text-zinc-300";
    case "errored":
      return "bg-red-500/10 text-red-700 dark:text-red-300";
    default:
      return "bg-zinc-500/10";
  }
}
