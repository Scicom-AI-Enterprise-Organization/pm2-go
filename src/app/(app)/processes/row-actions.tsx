"use client";

import { useTransition } from "react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { MoreHorizontal } from "lucide-react";
import {
  deleteAction,
  reloadAction,
  restartAction,
  stopAction,
} from "./actions";

type Props = {
  name: string;
  state: string;
  canWrite: boolean;
  canDelete: boolean;
};

export function ProcessRowActions({ name, state, canWrite, canDelete }: Props) {
  const [pending, start] = useTransition();
  if (!canWrite && !canDelete) return null;

  const run = (label: string, fn: () => Promise<unknown>) =>
    start(async () => {
      try {
        await fn();
        toast.success(`${label}: ${name}`);
      } catch (e) {
        toast.error(`${label} ${name} failed: ${(e as Error).message}`);
      }
    });

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" size="icon" disabled={pending} aria-label={`Actions for ${name}`}>
          <MoreHorizontal className="h-4 w-4" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        {canWrite ? (
          <>
            <DropdownMenuItem onSelect={() => run("Restarted", () => restartAction(name))}>
              Restart
            </DropdownMenuItem>
            <DropdownMenuItem onSelect={() => run("Reloaded", () => reloadAction(name))}>
              Reload (graceful)
            </DropdownMenuItem>
            <DropdownMenuItem
              onSelect={() => run("Stopped", () => stopAction(name))}
              disabled={state === "stopped"}
            >
              Stop
            </DropdownMenuItem>
          </>
        ) : null}
        {canDelete ? (
          <DropdownMenuItem
            className="text-red-600 focus:text-red-600 dark:text-red-400 dark:focus:text-red-400"
            onSelect={() => {
              if (confirm(`Delete ${name}? This stops and removes it from the dump.`)) {
                run("Deleted", () => deleteAction(name));
              }
            }}
          >
            Delete
          </DropdownMenuItem>
        ) : null}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
